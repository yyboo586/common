# Worker 执行流程深度分析

## 一、关键理解

### 1. Worker 与 TaskType 的关系

**每个 TaskType 对应一个 Worker goroutine**（不是每个任务一个 Worker）

```go
// Start 方法中
for taskType, handler := range m.handlers {
    m.sigChanMap[taskType] = make(chan struct{}, 1)
    m.wg.Add(1)
    go m.worker(taskType, handler)  // 每个 TaskType 启动一个 Worker
}
```

### 2. AddTask 的时机

```go
// dao.go: AddTask
func (d *DAO) AddTask(...) {
    data := g.Map{
        "next_retry_time": gtime.Now().Unix(),  // ← 立即执行
    }
}
```

### 3. Worker 的循环逻辑

```go
func (m *AsyncTaskManager) worker(taskType TaskType, handler TaskHandler) {
    nextFetchTime := time.Now()
    
    for {
        // 步骤1: 等待唤醒
        select {
        case <-m.sigChanMap[taskType]:        // 信号唤醒
        case <-time.After(time.Until(nextFetchTime)):  // 定时器唤醒
        }
        
        // 步骤2: 尝试获取任务（带时间条件）
        task, err := m.dao.FetchPendingTask(m.ctx, taskType)
        // SQL: WHERE task_type=X AND status=0 AND next_retry_time < 当前时间
        
        // 步骤3: 没有可执行任务
        if task == nil {
            minTask, err := m.dao.GetMinNextRetryTime(m.ctx, taskType)
            // SQL: WHERE task_type=X AND status=0 ORDER BY next_retry_time
            
            if minTask != nil {
                nextFetchTime = minTask.NextRetryTime  // 等到下次任务时间
            } else {
                nextFetchTime = time.Now().Add(30s)    // 没有任务，30秒后查询
            }
            continue  // ← 回到步骤1
        }
        
        // 步骤4: 处理任务
        m.handleTask(task, handler)
        
        // 步骤5: 立即查询下一个任务
        nextFetchTime = time.Now()  // ← 关键！立即查询
        // continue 回到步骤1
    }
}
```

## 二、正常场景分析

### 场景1：添加 5 个任务，调用 1 次 WakeUp

```
时刻 T0: 添加 5 个任务（task_type=6, next_retry_time=当前时间）
         调用 WakeUp(6) 一次
         
时刻 T1: Worker 被唤醒（channel 收到 1 个信号）
         FetchPendingTask → 获取任务1 ✓
         handleTask(任务1)
         nextFetchTime = Now()
         
时刻 T2: 回到 select
         time.After(0) 立即触发（因为 nextFetchTime = Now()）
         FetchPendingTask → 获取任务2 ✓
         handleTask(任务2)
         nextFetchTime = Now()
         
时刻 T3-T5: 同理，处理任务3、4、5
         
时刻 T6: FetchPendingTask → 没有任务（返回 nil）
         GetMinNextRetryTime → 查询下次时间
         设置 nextFetchTime（可能是未来时间或 Now()+30s）
         continue
         
时刻 T7: select 等待定时器...

总计查询：6 次 FetchPendingTask + 1 次 GetMinNextRetryTime = 7 次
```

**这是正常的！**

## 三、惊群场景分析

### 场景2：添加 5 个任务，调用 20 次 WakeUp

```
时刻 T0: 添加 5 个任务（task_type=6）
         业务代码调用 20 次 WakeUp(6)
         channel 中堆积 20 个信号（旧版本容量 1000）
         
时刻 T1: Worker 被唤醒（消费第 1 个信号）
         FetchPendingTask → 获取任务1 ✓
         handleTask(任务1)
         nextFetchTime = Now()
         
时刻 T2: 回到 select
         time.After(0) 立即触发
         FetchPendingTask → 获取任务2 ✓
         handleTask(任务2)
         
时刻 T3-T5: 处理任务3、4、5（正常）
         
时刻 T6: FetchPendingTask → 没有任务（nil）
         GetMinNextRetryTime → 查询到定时任务（未来时间）
         nextFetchTime = 未来时间（例如 1 小时后）
         continue
         
时刻 T7: 回到 select
         ⚠️ channel 中还有 19 个信号！
         select 立即收到第 2 个信号（不等待定时器）
         FetchPendingTask → 没有任务（nil）[rows:0]
         GetMinNextRetryTime → 查询到定时任务 [rows:1]
         continue
         
时刻 T8: 回到 select
         ⚠️ channel 中还有 18 个信号！
         select 立即收到第 3 个信号
         FetchPendingTask → 没有任务（nil）[rows:0]
         GetMinNextRetryTime → 查询到定时任务 [rows:1]
         continue
         
时刻 T9-T26: 重复上述过程（消费剩余 17 个信号）

总计查询：
  - 5 次 FetchPendingTask（成功）
  - 20 次 FetchPendingTask（失败）[rows:0]
  - 20 次 GetMinNextRetryTime [rows:1]
  = 45 次查询！
```

**这就是惊群的根本原因！**

## 四、问题的本质

### 关键发现

1. **Worker 处理完所有任务后，会设置 `nextFetchTime = 未来时间`**
   - 如果队列空了，应该等待到 nextFetchTime 再查询
   - 或者等待新的 WakeUp 信号

2. **但如果 channel 中堆积了多个信号**
   - select 会立即收到信号，不等待定时器
   - 即使没有可执行任务，也会重复查询

3. **为什么会堆积信号？**
   - 业务代码在短时间内多次调用 WakeUp（如您的代码中一次操作调用 3 次）
   - 旧版本 channel 容量为 1000，可以堆积很多信号

### 日志佐证

```
[rows:0] FetchPendingTask - 没有可执行任务（时间条件不满足）
[rows:1] GetMinNextRetryTime - 查到未来的定时任务
[rows:0] FetchPendingTask - 又查一次，还是没有
[rows:1] GetMinNextRetryTime - 又查一次，还是那个定时任务
...重复 N 次
```

这说明：
- 有定时任务存在（未来时间）
- Worker 被反复唤醒，每次都查询，但都因时间条件不满足而失败
- 原因是 channel 中堆积了大量信号

## 五、修复方案的正确性

### 修复1：Channel 容量改为 1

```go
m.sigChanMap[taskType] = make(chan struct{}, 1)
```

**效果**：无论调用多少次 WakeUp，channel 最多只有 1 个信号。

### 修复2：清空堆积信号

```go
case <-m.sigChanMap[taskType]:
    // 清空所有堆积信号（理论上不会超过 1 个）
    for len(m.sigChanMap[taskType]) > 0 {
        <-m.sigChanMap[taskType]
    }
```

**效果**：即使有多个信号，一次性清空，只查询一次。

### 为什么这是安全的？

**关键：Worker 处理完一个任务后，会立即查询下一个**

```go
// 处理任务
m.handleTask(task, handler)
// 立即查询下一个
nextFetchTime = time.Now()  // ← 这保证了不会遗漏任务
```

**即使 WakeUp 信号被合并/丢失，Worker 也会：**
1. 处理完任务1 → 立即查询任务2
2. 处理完任务2 → 立即查询任务3
3. ...直到队列为空

**唯一的例外：第一个任务**
- 第一个任务需要 WakeUp 来唤醒 Worker
- 但 channel 容量为 1，第一个 WakeUp 一定会成功
- 后续的 WakeUp 即使失败也不影响（Worker 会持续处理）

## 六、您的质疑的回应

### 质疑1："添加几个任务，就应该唤醒几个对应的 Worker"

**回应**：理解有偏差

- 不是"添加几个任务就唤醒几个 Worker"
- 而是"每个 TaskType 有且只有一个 Worker"
- 添加 N 个相同 TaskType 的任务，只有 1 个 Worker 串行处理

**示例**：
```go
// 添加 5 个 TaskType=6 的任务
AddTask(type=6, id=1)
AddTask(type=6, id=2)
AddTask(type=6, id=3)
AddTask(type=6, id=4)
AddTask(type=6, id=5)

WakeUp(type=6)  // 只需要调用 1 次！

// TaskType=6 的 Worker 会：
// 1. 被唤醒
// 2. 处理任务1
// 3. 立即查询任务2（不需要再次 WakeUp）
// 4. 处理任务2
// 5. ...依次处理完所有任务
```

### 质疑2："AddTask 时 NextRetryTime = Now()，应该立即处理"

**回应**：完全正确！

但问题不在这里，问题在于：
- **任务已经处理完了**
- **但 channel 中还有 19 个信号**
- **Worker 被反复唤醒，继续查询（即使已经没任务了）**

## 七、结论

1. **惊群的根本原因**：WakeUp 信号堆积，导致任务处理完后仍被反复唤醒
2. **修复是必要的**：清空堆积信号，避免无效查询
3. **修复是安全的**：Worker 会持续处理任务，不会因为信号合并而遗漏任务
4. **性能提升显著**：查询次数从 40+ 次降低到 7 次左右

---

**您的理解是对的**：AddTask 后应该立即处理。  
**我的修复也是对的**：避免因信号堆积导致的重复查询。

两者不冲突！
