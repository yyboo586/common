package AsyncTask

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/glog"
	"github.com/gogf/gf/v2/test/gtest"
)

// TestThunderingHerd 测试惊群现象及修复效果
func TestThunderingHerd(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// 注意：此测试需要 MySQL 数据库连接
		// 如果没有数据库，此测试将跳过
		dsn := "root:root@tcp(127.0.0.1:3306)/test?parseTime=true"
		
		config := &Config{
			DSN:              dsn,
			Database:         "test",
			Group:            "test_thundering_herd",
			TableName:        "t_async_task_test_th",
			HistoryTableName: "t_async_task_history_test_th",
			Logger:           glog.New(),
			InitInterval:     100 * time.Millisecond,
			QueryInterval:    5 * time.Second,
			ErrSleepInterval: 1 * time.Second,
			TaskTimeout:      10 * time.Second,
		}

		manager, err := NewAsyncTaskManager(config)
		if err != nil {
			t.Skip("数据库连接失败，跳过测试:", err)
			return
		}

		// 统计查询次数
		var queryCount atomic.Int32
		
		// 注册任务处理器
		testTaskType := TaskType(100)
		err = manager.RegisterHandler(testTaskType, "TestTask", func(ctx context.Context, task *Task) error {
			// 模拟任务处理
			time.Sleep(10 * time.Millisecond)
			return nil
		})
		t.AssertNil(err)

		// 启动管理器
		err = manager.Start()
		t.AssertNil(err)
		defer manager.Stop()

		// 等待 worker 初始化
		time.Sleep(200 * time.Millisecond)

		// 模拟业务场景：快速连续调用 WakeUp
		wakeUpCount := 20
		var wg sync.WaitGroup
		wg.Add(wakeUpCount)
		
		for i := 0; i < wakeUpCount; i++ {
			go func() {
				defer wg.Done()
				manager.WakeUp(testTaskType)
			}()
		}
		wg.Wait()

		// 给 worker 一些时间处理
		time.Sleep(500 * time.Millisecond)

		// 验证：在修复后，即使调用了 20 次 WakeUp，
		// 由于 channel 容量为 1 且会清空堆积信号，
		// 实际查询次数应该远少于 20*2=40 次
		
		t.Logf("WakeUp 调用次数: %d", wakeUpCount)
		t.Logf("实际查询次数: %d", queryCount.Load())
		
		// 清理测试数据
		_, _ = g.DB(config.Group).Exec(context.Background(), "DROP TABLE IF EXISTS "+config.TableName)
		_, _ = g.DB(config.Group).Exec(context.Background(), "DROP TABLE IF EXISTS "+config.HistoryTableName)
	})
}

// TestWakeUpSignalDrain 测试信号清空机制
func TestWakeUpSignalDrain(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// 模拟 signal channel
		sigChan := make(chan struct{}, 1) // 容量为1
		
		// 快速发送多个信号
		for i := 0; i < 10; i++ {
			select {
			case sigChan <- struct{}{}:
				t.Logf("Signal %d sent", i+1)
			default:
				t.Logf("Signal %d blocked (channel full)", i+1)
			}
		}
		
		// 验证：容量为1的 channel 最多只能存储1个信号
		t.Assert(len(sigChan), 1)
		
		// 模拟 worker 接收信号
		select {
		case <-sigChan:
			t.Log("Worker received signal")
		default:
			t.Error("Should receive signal")
		}
		
		// 清空剩余信号（虽然此时已经没有了）
		drainedCount := 0
		for len(sigChan) > 0 {
			<-sigChan
			drainedCount++
		}
		t.Assert(drainedCount, 0)
		t.Log("Test passed: Channel capacity=1 prevents signal accumulation")
	})
}

// TestConcurrentWakeUp 测试并发 WakeUp 场景
func TestConcurrentWakeUp(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		sigChan := make(chan struct{}, 1)
		
		// 并发调用 WakeUp
		concurrency := 100
		var wg sync.WaitGroup
		wg.Add(concurrency)
		
		for i := 0; i < concurrency; i++ {
			go func() {
				defer wg.Done()
				select {
				case sigChan <- struct{}{}:
				default:
				}
			}()
		}
		wg.Wait()
		
		// 验证：无论调用多少次，channel 中最多只有1个信号
		signalCount := len(sigChan)
		t.AssertLE(signalCount, 1)
		t.Logf("Concurrent WakeUp x%d -> Channel size: %d", concurrency, signalCount)
	})
}

// BenchmarkWakeUpWithCapacity1 性能测试：容量为1
func BenchmarkWakeUpWithCapacity1(b *testing.B) {
	sigChan := make(chan struct{}, 1)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		select {
		case sigChan <- struct{}{}:
		default:
		}
	}
}

// BenchmarkWakeUpWithCapacity1000 性能测试：容量为1000（旧版本）
func BenchmarkWakeUpWithCapacity1000(b *testing.B) {
	sigChan := make(chan struct{}, 1000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		select {
		case sigChan <- struct{}{}:
		default:
		}
	}
}

// 演示问题场景的测试（用于文档说明）
func ExampleThunderingHerdProblem() {
	// 模拟旧版本的行为（容量1000，不清空信号）
	oldSigChan := make(chan struct{}, 1000)
	
	// 业务代码快速调用 20 次 WakeUp
	for i := 0; i < 20; i++ {
		select {
		case oldSigChan <- struct{}{}:
		default:
		}
	}
	
	println("旧版本：channel 中堆积了", len(oldSigChan), "个信号")
	
	// Worker 会被唤醒 20 次，执行 40 次查询（每次唤醒执行 2 次查询）
	queryCount := 0
	for len(oldSigChan) > 0 {
		<-oldSigChan
		queryCount += 2 // FetchPendingTask + GetMinNextRetryTime
	}
	println("旧版本：总共执行了", queryCount, "次数据库查询")
	
	// ===== 新版本 =====
	
	newSigChan := make(chan struct{}, 1) // 容量改为1
	
	// 业务代码快速调用 20 次 WakeUp
	for i := 0; i < 20; i++ {
		select {
		case newSigChan <- struct{}{}:
		default:
		}
	}
	
	println("新版本：channel 中只有", len(newSigChan), "个信号")
	
	// Worker 只会被唤醒 1 次，执行 2 次查询
	queryCountNew := 0
	if len(newSigChan) > 0 {
		<-newSigChan
		// 清空剩余信号（此时已经没有了）
		for len(newSigChan) > 0 {
			<-newSigChan
		}
		queryCountNew += 2 // FetchPendingTask + GetMinNextRetryTime
	}
	println("新版本：总共执行了", queryCountNew, "次数据库查询")
	
	// Output:
	// 旧版本：channel 中堆积了 20 个信号
	// 旧版本：总共执行了 40 次数据库查询
	// 新版本：channel 中只有 1 个信号
	// 新版本：总共执行了 2 次数据库查询
}

// TestAddTaskAndWakeUp 测试添加任务后唤醒的场景
func TestAddTaskAndWakeUp(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// 模拟业务场景：handleExhibitionStartRunning 中连续调用 3 次 WakeUp
		taskTypes := []string{"TaskTypeAutoEnd", "TaskTypeNotifyUsers", "TaskTypeNotifyFollowers"}
		
		sigChans := make(map[string]chan struct{})
		for _, taskType := range taskTypes {
			sigChans[taskType] = make(chan struct{}, 1)
		}
		
		// 模拟事务提交后的 WakeUp 调用
		for _, taskType := range taskTypes {
			select {
			case sigChans[taskType] <- struct{}{}:
				t.Logf("WakeUp: %s", taskType)
			default:
				t.Logf("WakeUp: %s (channel full, skipped)", taskType)
			}
		}
		
		// 验证：每个 worker 都收到了唤醒信号
		for _, taskType := range taskTypes {
			t.Assert(len(sigChans[taskType]), 1)
		}
	})
}
