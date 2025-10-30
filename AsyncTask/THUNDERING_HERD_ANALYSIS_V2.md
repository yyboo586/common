# AsyncTask 惊群现象分析（单例部署）

## 问题重现

日志显示：在 11 毫秒内（17:41:01.969-980），task_type=6 和 task_type=7 的查询被重复执行了 40-50 次。

## 根本原因：WakeUp 信号堆积导致的连续查询

### 问题核心机制

#### 1. Worker 的循环逻辑

```go
func (m *AsyncTaskManager) worker(taskType TaskType, handler TaskHandler) {
    for {
        // 等待信号或定时器
        select {
        case <-m.sigChanMap[taskType]: // 收到唤醒信号，消费1个
        case <-time.After(time.Until(nextFetchTime)):
        }
        
        // 查询1：获取待处理任务
        task, err := m.dao.FetchPendingTask(m.ctx, taskType)
        
        // 没有待处理任务
        if task == nil {
            // 查询2：获取下次执行时间
            minTask, err := m.dao.GetMinNextRetryTime(m.ctx, taskType)
            // ...
            continue  // ← 关键：回到 select，如果 channel 有信号，立即被唤醒
        }
        
        // 处理任务
        m.handleTask(task, handler)
        nextFetchTime = time.Now()  // 立即查询下一个
    }
}
```

#### 2. WakeUp 的实现

```go
func (m *AsyncTaskManager) wakeUp(taskType TaskType) {
    select {
    case ch <- struct{}{}: // 往 channel 发送信号
    default:
        // 通道已满（容量1000）
    }
}
```

Channel 容量（Start 方法中）：
```go
m.sigChanMap[taskType] = make(chan struct{}, 1000)
```

### 惊群发生流程

假设 channel 中堆积了 20 个 WakeUp 信号：

```
时刻 T0: Channel 状态: [信号1, 信号2, 信号3, ..., 信号20]

时刻 T1: Worker select 收到信号1
         ↓
         执行 FetchPendingTask (查询1) → 没有可执行任务
         ↓
         执行 GetMinNextRetryTime (查询2) → 设置 nextFetchTime
         ↓
         continue 回到 select

时刻 T2: select 发现 channel 中还有信号2，立即返回（不等待）
         ↓
         再次执行 FetchPendingTask (查询3)
         ↓
         再次执行 GetMinNextRetryTime (查询4)
         ↓
         continue

时刻 T3-T20: 重复上述过程...
```

**关键点**：`select` 语句在检查多个 case 时，如果 channel 有数据，会**立即**返回，不会等待定时器。

### 为什么会堆积大量信号？

#### 情况1：业务代码中的多次 WakeUp 调用

从您提供的业务代码中，`handleExhibitionStartRunning` 方法：

```go
func (e *exhibition) handleExhibitionStartRunning(...) {
    // ... 事务提交成功后 ...
    
    // 连续调用 3 次 WakeUp
    e.asyncTask.WakeUp(model.TaskTypeExhibitionAutoEnd)
    e.asyncTask.WakeUp(model.TaskTypeExStartAndNotifyReservationUsers)
    e.asyncTask.WakeUp(model.TaskTypeCheckAndNotifySPFollowersOfNewExhibition)
    
    return nil
}
```

**每次调用 WakeUp 都会往 channel 发送 1 个信号。**

#### 情况2：批量操作或测试场景

如果有批量审批展会的操作：

```go
// 假设批量审批 10 个展会
for i := 0; i < 10; i++ {
    HandleEvent(ctx, exhibitionID, ExhibitionEventApprove, data)
    // 每次审批都会调用：
    // e.asyncTask.WakeUp(model.TaskTypeExhibitionAutoStartEnrolling)
}
```

这会在短时间内往同一个 channel 发送 10 个信号。

#### 情况3：任务自身触发 WakeUp

观察业务流程：

```
Approve (审核通过)
  ↓ 添加定时任务：TaskTypeExhibitionAutoStartEnrolling
  ↓ WakeUp(TaskTypeExhibitionAutoStartEnrolling)  ← 信号1
  
Worker 被唤醒，处理任务：AutoStartEnrolling
  ↓ 触发 StartEnrolling 事件
  ↓ 添加定时任务：TaskTypeExhibitionAutoEndEnrolling
  ↓ WakeUp(TaskTypeExhibitionAutoEndEnrolling)     ← 信号2
  ↓ WakeUp(TaskTypeExhibitionAutoEndEnrolling)     ← 可能因为某些原因重复调用
```

### 从日志验证

查看日志中的 task_type=6 和 task_type=7：

```
同一毫秒内：
- task_type=6: [rows:1] → 查询到任务（但可能 next_retry_time 不满足）
- task_type=6: [rows:0] → 时间条件不满足，没有可执行任务
- task_type=6: [rows:1] → 再次查询到任务
- task_type=6: [rows:0] → 再次时间条件不满足
...重复 20+ 次
```

**这说明**：
1. 数据库中确实有 task_type=6 的任务（status=0）
2. 但 `next_retry_time` 是未来时间（定时任务）
3. Worker 被反复唤醒，每次都查询，但都因为时间条件不满足而返回 nil
4. 然后 `GetMinNextRetryTime` 查询到这个未来任务，设置 nextFetchTime
5. 但由于 channel 中还有信号，立即再次被唤醒，重复查询

### 核心问题总结

**设计缺陷**：Worker 没有在被唤醒后**清空 channel 中的所有信号**，而是每次只消费 1 个信号。

```go
// 当前逻辑（有问题）
select {
case <-sigChan:  // 只消费 1 个信号
    // 处理...
}

// 期望逻辑（优化后）
select {
case <-sigChan:
    // 消费所有堆积的信号
    for len(sigChan) > 0 {
        <-sigChan
    }
    // 然后处理...
}
```

## 为什么设计时说"最多查询两次"？

设计假设：
- **每个任务只会触发 1 次 WakeUp**
- Worker 被唤醒后，执行 1 次 FetchPendingTask
- 如果没有可执行任务，执行 1 次 GetMinNextRetryTime
- 共 2 次查询

实际情况：
- **1 个任务可能触发多次 WakeUp**（业务代码中有 3 次）
- **多个任务在短时间内创建**，每个都触发 WakeUp
- **Channel 中堆积了 N 个信号**
- Worker 被唤醒 N 次，执行 2N 次查询

## 解决方案

### 方案1：清空信号队列（推荐）

修改 worker 方法：

```go
func (m *AsyncTaskManager) worker(taskType TaskType, handler TaskHandler) {
    for {
        select {
        case <-m.sigChanMap[taskType]:
            // 清空 channel 中的所有堆积信号
            for len(m.sigChanMap[taskType]) > 0 {
                <-m.sigChanMap[taskType]
            }
        case <-time.After(time.Until(nextFetchTime)):
        case <-m.ctx.Done():
            return
        }
        
        // 执行查询和处理（只执行一次）
        // ...
    }
}
```

### 方案2：优化 WakeUp 逻辑

将多次 WakeUp 合并为 1 次：

```go
// 业务代码修改
func (e *exhibition) handleExhibitionStartRunning(...) {
    // ... 事务提交成功后 ...
    
    // 方式1：只在最后调用一次 WakeUp
    taskTypes := []TaskType{
        model.TaskTypeExhibitionAutoEnd,
        model.TaskTypeExStartAndNotifyReservationUsers,
        model.TaskTypeCheckAndNotifySPFollowersOfNewExhibition,
    }
    
    // 去重后唤醒
    for _, taskType := range unique(taskTypes) {
        e.asyncTask.WakeUp(taskType)
    }
}
```

### 方案3：使用布尔标志替代计数

修改 sigChanMap 为 bool channel（容量1）：

```go
// 初始化
m.sigChanMap[taskType] = make(chan struct{}, 1) // 容量改为 1

// WakeUp 逻辑不变（已经是非阻塞）
select {
case ch <- struct{}{}:
default: // channel 满了，说明已有信号，无需重复发送
}
```

这样 channel 中最多只有 1 个信号，避免堆积。

### 方案4：添加节流机制

在 worker 中添加最小查询间隔：

```go
func (m *AsyncTaskManager) worker(taskType TaskType, handler TaskHandler) {
    lastQueryTime := time.Time{}
    minQueryInterval := 100 * time.Millisecond // 最小查询间隔
    
    for {
        select {
        case <-m.sigChanMap[taskType]:
            // 如果距离上次查询时间太短，忽略
            if time.Since(lastQueryTime) < minQueryInterval {
                continue
            }
        case <-time.After(time.Until(nextFetchTime)):
        }
        
        lastQueryTime = time.Now()
        // 执行查询...
    }
}
```

## 推荐方案组合

1. **方案1（清空信号队列）** - 根本解决问题
2. **方案3（容量改为1）** - 防止信号堆积
3. **调整业务代码** - 避免不必要的重复 WakeUp

## 验证方法

修改后，在日志中添加：

```go
func (m *AsyncTaskManager) worker(taskType TaskType, handler TaskHandler) {
    for {
        select {
        case <-m.sigChanMap[taskType]:
            drainedCount := 0
            for len(m.sigChanMap[taskType]) > 0 {
                <-m.sigChanMap[taskType]
                drainedCount++
            }
            if drainedCount > 0 {
                m.logger.Infof(ctx, "[AsyncTask] [%s] Drained %d signals", 
                    m.getTaskTypeText(taskType), drainedCount)
            }
        case <-time.After(time.Until(nextFetchTime)):
        }
        
        // ...查询和处理
    }
}
```

如果日志输出 "Drained 20 signals"，就证明了问题原因。
