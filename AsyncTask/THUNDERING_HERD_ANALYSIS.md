# AsyncTask 惊群现象分析

## 问题描述

从日志中可以看到，在短短 11 毫秒内（17:41:01.969-980），相同的查询被重复执行了数十次：

```sql
-- 查询1：无时间限制（GetMinNextRetryTime）
SELECT ... FROM t_async_task WHERE (task_type=6) AND (status=0) ORDER BY next_retry_time ASC LIMIT 1

-- 查询2：有时间限制（FetchPendingTask）  
SELECT ... WHERE (task_type=6) AND (status=0) AND (next_retry_time < 1761817261) ORDER BY next_retry_time ASC LIMIT 1
```

## 设计预期 vs 实际情况

**设计预期**：一个待处理任务最多查询两次
1. 第一次：`FetchPendingTask` 查询可执行的任务（有时间条件）
2. 第二次：如果没有可执行任务，`GetMinNextRetryTime` 查询下次最早执行时间（无时间条件）

**实际情况**：每个任务被查询了数十次

## 根本原因：多实例部署的惊群效应

### 1. 多实例并发查询

从日志查询次数（40-50次）推测，系统采用了**多实例部署**（如 Kubernetes 多 Pod、多服务器）：

```
实例1 Worker (task_type=6) ──┐
实例2 Worker (task_type=6) ──┤
实例3 Worker (task_type=6) ──┤
...                           ├──> 同时查询同一张表
实例N Worker (task_type=6) ──┘
```

### 2. Worker 轮询机制

查看 `async_task_manager.go` 中的 `worker` 方法（218-288行）：

```go
func (m *AsyncTaskManager) worker(taskType TaskType, handler TaskHandler) {
    // ...
    for {
        // 等待信号或定时器
        select {
        case <-m.sigChanMap[taskType]: // 唤醒信号
        case <-time.After(time.Until(nextFetchTime)): // 定时触发
        }
        
        // 获取待处理任务 (查询1)
        task, err := m.dao.FetchPendingTask(m.ctx, taskType)
        
        // 没有待处理任务
        if task == nil {
            // 查询下次执行时间 (查询2)
            minTask, err := m.dao.GetMinNextRetryTime(m.ctx, taskType)
            // ...
        }
    }
}
```

### 3. 乐观锁冲突

`FetchPendingTask` 使用乐观锁机制（dao.go:142-189）：

```go
func (d *DAO) FetchPendingTask(ctx context.Context, taskType TaskType) (out *Task, err error) {
    // 第1步：查询待处理任务
    err = d.db.Model(d.tableName).Ctx(ctx).
        Where("task_type", int(taskType)).
        Where("status", int(TaskStatusPending)).
        WhereLT("next_retry_time", gtime.Now().Unix()).
        OrderAsc("next_retry_time").
        Scan(&entity)
    
    // 第2步：尝试更新状态为Processing（乐观锁）
    result, err := d.db.Model(d.tableName).Ctx(ctx).
        Where("id", entity.ID).
        Where("version", entity.Version). // 乐观锁：版本号必须匹配
        Data(g.Map{
            "status":      int(TaskStatusProcessing),
            "version":     entity.Version + 1,
            "update_time": gtime.Now().Unix(),
        }).
        Update()
    
    // 检查更新结果
    if rowsAffected == 0 {
        return nil, ErrNoRowsAffected // 乐观锁冲突，其他实例已抢到任务
    }
}
```

## 惊群发生流程

以 2 个实例、1 个待处理任务为例：

```
时刻 T0: 任务创建（id=1, version=0, status=Pending）

时刻 T1: 
  实例A Worker: SELECT ... [查询到 task id=1, version=0]
  实例B Worker: SELECT ... [查询到 task id=1, version=0]
  
时刻 T2:
  实例A Worker: UPDATE ... WHERE id=1 AND version=0 → 成功 (rowsAffected=1)
  实例B Worker: UPDATE ... WHERE id=1 AND version=0 → 失败 (rowsAffected=0, version已变成1)
  
时刻 T3:
  实例A Worker: 处理任务
  实例B Worker: 返回 ErrNoRowsAffected → 进入下一轮循环 → 再次查询
```

**关键问题**：当有 N 个实例时，每个任务都会被查询 N 次（SELECT），但只有 1 个实例能成功获取（UPDATE）。

## 日志佐证

```
[rows:1] - 查询到任务（但可能后续乐观锁失败）
[rows:0] - 未查询到任务（已被其他实例处理，或时间条件不满足）
```

大量的 `[rows:0]` 说明：
- 多个实例同时查询
- 任务已被其他实例通过乐观锁抢占处理
- 后续实例查询时任务状态已变为 Processing 或 Success

## 性能影响

1. **数据库压力**：N 个实例 × 每秒查询频率 = 大量无效查询
2. **CPU 浪费**：大部分查询都是无效的（乐观锁冲突）
3. **网络开销**：实例与数据库间的重复通信

## 解决方案建议

### 方案1：分布式锁（推荐）

在查询前先获取分布式锁：

```go
func (d *DAO) FetchPendingTask(ctx context.Context, taskType TaskType) (out *Task, err error) {
    // 使用 Redis 分布式锁
    lockKey := fmt.Sprintf("async_task:lock:%d", taskType)
    lock := redis.TryLock(lockKey, 10*time.Second)
    if lock == nil {
        return nil, ErrLockFailed // 其他实例正在处理，直接返回
    }
    defer lock.Unlock()
    
    // 原有的查询和更新逻辑...
}
```

### 方案2：任务分片

给每个实例分配不同的任务范围：

```go
// 根据实例ID和总实例数进行哈希分片
func (d *DAO) FetchPendingTask(ctx context.Context, taskType TaskType, instanceID, totalInstances int) {
    err = d.db.Model(d.tableName).Ctx(ctx).
        Where("task_type", int(taskType)).
        Where("status", int(TaskStatusPending)).
        Where("id % ? = ?", totalInstances, instanceID). // 按ID取模分片
        WhereLT("next_retry_time", gtime.Now().Unix()).
        OrderAsc("next_retry_time").
        Scan(&entity)
}
```

### 方案3：增加查询间隔

减少无效查询频率：

```go
// 当发生乐观锁冲突时，增加退避时间
if err == ErrNoRowsAffected {
    nextFetchTime = time.Now().Add(m.config.ErrSleepInterval * 2) // 从3秒增加到6秒
    continue
}
```

### 方案4：事件驱动（最优）

使用消息队列替代轮询：

```
新任务添加 → 发送到 MQ → 消费者竞争消费 → 自然负载均衡
```

## 总结

当前的惊群问题主要由**多实例部署 + 轮询机制 + 乐观锁**共同导致：
- 多实例同时轮询同一张表
- 乐观锁虽然保证了并发安全，但无法避免重复查询
- 每个任务被查询 N 次（N = 实例数），但只有 1 次有效

建议优先采用**分布式锁**或**任务分片**方案，从根本上减少无效查询。
