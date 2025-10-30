# 惊群现象修复说明

## 问题摘要

在单例部署的环境下，AsyncTask 模块出现了"惊群"现象：**短时间内（11毫秒）相同的 SQL 查询被执行了 40-50 次**，远超设计预期的"每个任务最多查询两次"。

## 根本原因

### 1. WakeUp 信号堆积机制

旧版本的 signal channel 容量为 **1000**：

```go
m.sigChanMap[taskType] = make(chan struct{}, 1000)  // 旧版本
```

当业务代码在短时间内多次调用 `WakeUp()` 时，信号会在 channel 中堆积。

### 2. Worker 逐个消费信号

Worker 的 select 语句每次只消费 1 个信号：

```go
select {
case <-m.sigChanMap[taskType]:  // 只消费 1 个
    // 执行查询...
    // 回到 select，如果 channel 中还有信号，立即再次被唤醒
}
```

### 3. 业务代码触发场景

从您提供的业务代码中，存在多处连续调用 `WakeUp()` 的场景：

```go
// handleExhibitionStartRunning 中
e.asyncTask.WakeUp(model.TaskTypeExhibitionAutoEnd)
e.asyncTask.WakeUp(model.TaskTypeExStartAndNotifyReservationUsers)
e.asyncTask.WakeUp(model.TaskTypeCheckAndNotifySPFollowersOfNewExhibition)
```

### 4. 惊群流程

```
时刻 T0: 业务代码快速调用 20 次 WakeUp
         ↓
         Channel 中堆积 20 个信号

时刻 T1: Worker select 收到第 1 个信号
         ↓
         执行查询 (FetchPendingTask + GetMinNextRetryTime)
         ↓
         continue 回到 select

时刻 T2: Channel 中还有 19 个信号，Worker 立即被唤醒
         ↓
         再次执行查询（第 3、4 次）
         
时刻 T3-T20: 重复上述过程...
              ↓
              总共执行了 40 次查询！
```

**关键点**：定时任务的 `next_retry_time` 通常是未来时间，`FetchPendingTask` 查询不到可执行任务，但会查到任务存在。Worker 被反复唤醒，每次都重复查询。

## 修复方案

### 方案1：减小 Channel 容量（防止信号堆积）

```go
// 修改前
m.sigChanMap[taskType] = make(chan struct{}, 1000)

// 修改后
m.sigChanMap[taskType] = make(chan struct{}, 1)  // 容量改为 1
```

**效果**：无论调用多少次 `WakeUp()`，channel 中最多只有 1 个信号。

### 方案2：清空堆积信号（兜底保护）

```go
select {
case <-m.sigChanMap[taskType]:
    // 新增：清空 channel 中所有堆积的信号
    drainedCount := 0
    for len(m.sigChanMap[taskType]) > 0 {
        <-m.sigChanMap[taskType]
        drainedCount++
    }
    if drainedCount > 0 {
        m.logger.Debugf(ctx, "[AsyncTask] Drained %d pending signals", drainedCount)
    }
case <-time.After(time.Until(nextFetchTime)):
}
```

**效果**：即使 channel 中有多个信号（理论上不会超过 1 个），也会一次性清空，避免重复唤醒。

## 修复效果对比

### 修复前（旧版本）

```
Channel 容量: 1000
业务调用 WakeUp 20 次
  ↓
Channel 中堆积 20 个信号
  ↓
Worker 被唤醒 20 次
  ↓
执行 40 次数据库查询（20 × 2）
```

### 修复后（新版本）

```
Channel 容量: 1
业务调用 WakeUp 20 次
  ↓
Channel 中只有 1 个信号（其余被丢弃）
  ↓
Worker 被唤醒 1 次
  ↓
执行 2 次数据库查询（1 × 2）
```

**查询次数减少：40 次 → 2 次（减少 95%）**

## 为什么这个修复是安全的？

### 1. WakeUp 是"提醒"而非"命令"

`WakeUp()` 的语义是"提醒 worker 检查是否有新任务"，而不是"必须执行一次查询"。

- 多次提醒合并为 1 次是安全的
- Worker 被唤醒后会持续处理所有可执行任务，直到队列为空

### 2. 即使信号丢失也不会有问题

Worker 有两种唤醒方式：
1. **信号触发**：`WakeUp()` 主动唤醒
2. **定时器触发**：最多等待 `nextFetchTime`（下次任务执行时间）

即使信号全部丢失，Worker 也会在定时器到期时自动查询。

### 3. 处理任务后会立即查询下一个

```go
// 处理任务
err = m.handleTask(task, handler)

// 立即查询下一个任务
nextFetchTime = time.Now()  // 不等待，立即查询
```

这确保了有多个任务时，Worker 会持续处理，不会遗漏。

## 代码变更清单

### 文件1：`async_task_manager.go`

**变更位置1**：Start() 方法（第 163 行）

```diff
- m.sigChanMap[taskType] = make(chan struct{}, 1000)
+ m.sigChanMap[taskType] = make(chan struct{}, 1) // 容量改为1，防止信号堆积
```

**变更位置2**：worker() 方法（第 245-254 行）

```diff
  select {
  case <-m.sigChanMap[taskType]:
+     // 清空 channel 中所有堆积的信号，避免重复查询
+     drainedCount := 0
+     for len(m.sigChanMap[taskType]) > 0 {
+         <-m.sigChanMap[taskType]
+         drainedCount++
+     }
+     if drainedCount > 0 {
+         m.logger.Debugf(m.ctx, "[AsyncTask] [%s] Drained %d pending signals", m.getTaskTypeText(taskType), drainedCount)
+     }
  case <-time.After(time.Until(nextFetchTime)):
  }
```

## 验证方法

### 1. 运行测试

```bash
cd /workspace/AsyncTask
go test -v -run TestWakeUpSignalDrain
go test -v -run TestConcurrentWakeUp
```

### 2. 查看日志

修复后，如果出现信号堆积（理论上不会），日志会输出：

```
[AsyncTask] [TaskType名称] Drained N pending signals
```

正常情况下，不应该看到这条日志（因为容量为 1，不会堆积）。

### 3. 监控数据库查询

修复后，相同的 SQL 查询不应该在短时间内重复执行几十次。

## 性能影响

### 优点

1. **数据库压力大幅降低**：查询次数减少 95%
2. **CPU 使用率降低**：减少无效的循环和查询
3. **内存占用减少**：Channel 容量从 1000 降至 1

### 缺点

**无明显缺点**。`WakeUp()` 调用的开销可以忽略不计（仅 1 次 select 操作）。

## 后续优化建议

### 1. 业务代码优化（可选）

如果多个任务属于同一个 TaskType，可以只调用一次 `WakeUp()`：

```go
// 优化前
e.asyncTask.WakeUp(taskType1)
e.asyncTask.WakeUp(taskType2)
e.asyncTask.WakeUp(taskType1)  // 重复

// 优化后
taskTypes := []TaskType{taskType1, taskType2}
for _, taskType := range unique(taskTypes) {
    e.asyncTask.WakeUp(taskType)
}
```

但这不是必须的，因为 Channel 容量为 1 已经解决了问题。

### 2. 添加监控指标（可选）

可以添加 Prometheus 指标统计：

- `async_task_wakeup_total`：WakeUp 调用次数
- `async_task_signal_drained_total`：清空信号次数
- `async_task_query_total`：数据库查询次数

用于监控和告警。

## 总结

**问题**：WakeUp 信号堆积导致 Worker 被反复唤醒，产生大量重复查询。

**根因**：Channel 容量过大（1000），Worker 逐个消费信号。

**修复**：
1. 将 Channel 容量改为 1（防止堆积）
2. 添加信号清空逻辑（兜底保护）

**效果**：查询次数减少 95%，数据库压力大幅降低。

**安全性**：修复完全向后兼容，不影响功能正确性。
