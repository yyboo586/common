package AsyncTask

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/glog"
)

// Manager 异步任务管理器
type AsyncTaskManager struct {
	logger *glog.Logger
	config *Config
	dao    *DAO
	ctx    context.Context
	cancel context.CancelFunc

	handlers      map[TaskType]TaskHandler
	taskTypeTexts map[TaskType]string // 任务类型文本缓存
	sigChanMap    map[TaskType]chan struct{}
	mutex         sync.RWMutex

	wg     sync.WaitGroup
	closed bool
}

// New 创建AsyncTask管理器
func NewAsyncTaskManager(config *Config) (Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	dao, err := newDAO(ctx, config)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create DAO: %w", err)
	}

	m := &AsyncTaskManager{
		config:        config,
		logger:        config.Logger,
		dao:           dao,
		ctx:           ctx,
		cancel:        cancel,
		handlers:      make(map[TaskType]TaskHandler),
		taskTypeTexts: make(map[TaskType]string),
		sigChanMap:    make(map[TaskType]chan struct{}),
	}

	return m, nil
}

// EnsureTable 确保数据表存在
func (m *AsyncTaskManager) EnsureTable() error {
	return m.dao.EnsureTable()
}

// RegisterHandlerWithText 注册任务处理器（带任务类型文本）
func (m *AsyncTaskManager) RegisterHandler(taskType TaskType, taskTypeText string, handler TaskHandler) error {
	m.mutex.Lock()

	if m.closed {
		m.mutex.Unlock()
		return ErrManagerClosed
	}

	if _, ok := m.handlers[taskType]; ok {
		m.mutex.Unlock()
		return ErrHandlerAlreadyRegistered
	}

	m.handlers[taskType] = handler
	if taskTypeText != "" {
		m.taskTypeTexts[taskType] = taskTypeText
	} else {
		// 如果没有提供文本，使用默认格式
		m.taskTypeTexts[taskType] = fmt.Sprintf("TaskType[%d]", taskType)
	}
	m.mutex.Unlock()

	m.logger.Infof(m.ctx, "[AsyncTask] Handler registered for task: %s", m.getTaskTypeText(taskType))
	return nil
}

// getTaskTypeText 获取任务类型文本
func (m *AsyncTaskManager) getTaskTypeText(taskType TaskType) string {
	m.mutex.RLock()

	if text, ok := m.taskTypeTexts[taskType]; ok {
		m.mutex.RUnlock()
		return text
	}
	m.mutex.RUnlock()

	return fmt.Sprintf("TaskType[%d]", taskType)
}

// AddTaskWithTx 添加即时任务（支持事务）
func (m *AsyncTaskManager) AddTask(ctx context.Context, tx gdb.TX, taskType TaskType, customID string, content []byte) error {
	if tx == nil {
		return gerror.New("AddTask: tx is nil")
	}

	if m.closed {
		return gerror.New("AddTask: manager is closed")
	}

	err := m.dao.AddTask(ctx, tx, taskType, customID, content)
	if err != nil {
		return gerror.Wrap(err, "AddTask: failed to add task")
	}

	return nil
}

// AddScheduledTask 添加定时任务
func (m *AsyncTaskManager) AddScheduledTask(ctx context.Context, tx gdb.TX, taskType TaskType, customID string, content []byte, scheduledTime time.Time) error {
	if tx == nil {
		return gerror.New("AddScheduledTask: tx is nil")
	}

	if m.closed {
		return gerror.New("AddScheduledTask: manager is closed")
	}

	err := m.dao.AddScheduledTask(ctx, tx, taskType, customID, content, scheduledTime)
	if err != nil {
		return gerror.Wrap(err, "AddScheduledTask: failed to add scheduled task")
	}

	return nil
}

// Start 启动异步任务处理
func (m *AsyncTaskManager) Start() error {
	if err := m.EnsureTable(); err != nil {
		return gerror.Wrap(err, "Start: failed to ensure table")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.closed {
		return gerror.New("Start: manager is closed")
	}

	if len(m.handlers) == 0 {
		return gerror.New("Start: no handlers registered")
	}

	// 为每个任务类型启动工作线程
	for taskType, handler := range m.handlers {
		m.sigChanMap[taskType] = make(chan struct{}, 1000)
		m.wg.Add(1)
		go m.worker(taskType, handler)
	}

	// 启动超时监控
	m.wg.Add(1)
	go m.timeoutMonitor()

	m.logger.Infof(m.ctx, "[AsyncTask] Started with %d handler(s)", len(m.handlers))

	return nil
}

// Stop 停止异步任务处理
func (m *AsyncTaskManager) Stop() {
	m.mutex.Lock()
	if m.closed {
		m.mutex.Unlock()
		return
	}
	m.closed = true
	m.mutex.Unlock()

	m.cancel()
	m.wg.Wait()

	m.logger.Info(m.ctx, "[AsyncTask] Stopped. All workers have been stopped.")
}

// WakeUp 唤醒指定任务类型的工作线程
func (m *AsyncTaskManager) WakeUp(taskType TaskType) {
	m.wakeUp(taskType)
}

// wakeUp 内部唤醒方法
func (m *AsyncTaskManager) wakeUp(taskType TaskType) {
	m.mutex.RLock()
	ch, ok := m.sigChanMap[taskType]
	m.mutex.RUnlock()

	if !ok {
		m.logger.Errorf(m.ctx, "[AsyncTask] [%s] Signal channel not found", m.getTaskTypeText(taskType))
		return
	}

	select {
	case ch <- struct{}{}:
		// 信号发送成功
	default:
		// 通道已满
		m.logger.Warningf(m.ctx, "[AsyncTask] [%s] Signal channel is full", m.getTaskTypeText(taskType))
	}
}

// worker 工作线程
func (m *AsyncTaskManager) worker(taskType TaskType, handler TaskHandler) {
	defer m.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			m.logger.Errorf(m.ctx, "[AsyncTask] [%s] Worker panic: %v", m.getTaskTypeText(taskType), r)
		}
	}()

	// 初始化延迟
	time.Sleep(m.config.InitInterval)

	m.logger.Infof(m.ctx, "[AsyncTask] [%s] Worker started", m.getTaskTypeText(taskType))

	nextFetchTime := time.Now()

	for {
		// 检查退出信号
		select {
		case <-m.ctx.Done():
			m.logger.Infof(m.ctx, "[AsyncTask] [%s] Worker stopped", m.getTaskTypeText(taskType))
			return
		default:
		}

		// 等待信号或定时器
		select {
		case <-m.sigChanMap[taskType]: // 收到唤醒信号
		case <-time.After(time.Until(nextFetchTime)): // 定时器触发
		case <-m.ctx.Done():
			return
		}

		// 获取待处理任务
		task, err := m.dao.FetchPendingTask(m.ctx, taskType)
		if err != nil {
			if err != ErrNoRowsAffected {
				m.logger.Errorf(m.ctx, "[AsyncTask] [%s] Failed to fetch task: %v", m.getTaskTypeText(taskType), err)
			}
			nextFetchTime = time.Now().Add(m.config.ErrSleepInterval)
			continue
		}

		// 没有待处理任务
		if task == nil {
			// 查询下次执行时间
			minTask, err := m.dao.GetMinNextRetryTime(m.ctx, taskType)
			if err != nil {
				m.logger.Errorf(m.ctx, "[AsyncTask] [%s] Failed to get min retry time: %v", m.getTaskTypeText(taskType), err)
				nextFetchTime = time.Now().Add(m.config.ErrSleepInterval)
				continue
			}

			if minTask != nil {
				nextFetchTime = minTask.NextRetryTime
			} else {
				nextFetchTime = time.Now().Add(m.config.QueryInterval)
			}
			continue
		}

		// 处理任务
		err = m.handleTask(task, handler)
		if err != nil {
			m.logger.Errorf(m.ctx, "[AsyncTask] [%s] Failed to handle task (id: %d): %v", m.getTaskTypeText(taskType), task.ID, err)
		}

		// 立即查询下一个任务
		nextFetchTime = time.Now()
	}
}

// handleTask 处理任务
func (m *AsyncTaskManager) handleTask(task *Task, handler TaskHandler) error {
	ctx, cancel := context.WithTimeout(m.ctx, m.config.TaskTimeout)
	defer cancel()

	// 记录开始时间
	startTime := time.Now()
	startTimeUnix := startTime.UnixMilli()

	// 执行处理器
	err := handler(ctx, task)

	// 记录结束时间
	endTime := time.Now()
	endTimeUnix := endTime.UnixMilli()
	duration := endTime.Sub(startTime).Milliseconds()

	var status TaskStatus
	var nextRetryTime int64
	var lastError string
	var historyStatus int
	var result string

	if err != nil {
		// 处理失败，计算下次重试时间
		status = TaskStatusPending
		nextRetryTime = m.calculateNextRetryTime(task)
		lastError = err.Error()
		historyStatus = 0
		result = err.Error()
		m.logger.Debugf(ctx, "[AsyncTask] [%s] Task failed (id: %d, retry: %d): %v", m.getTaskTypeText(task.TaskType), task.ID, task.RetryCount, err)
	} else {
		// 处理成功
		status = TaskStatusSuccess
		nextRetryTime = 0
		lastError = "" // 成功时清空错误信息
		historyStatus = 1
		result = "success"
		m.logger.Debugf(ctx, "[AsyncTask] [%s] Task succeeded (id: %d)", m.getTaskTypeText(task.TaskType), task.ID)
	}

	// 更新任务状态
	updateErr := m.dao.UpdateTaskStatus(ctx, task, status, nextRetryTime, lastError)
	if updateErr != nil {
		m.logger.Errorf(ctx, "[AsyncTask] [%s] Failed to update task status (id: %d): %v", m.getTaskTypeText(task.TaskType), task.ID, updateErr.Error())
		return updateErr
	}

	// 记录执行历史
	// round 表示第几次执行（retry_count + 1 表示当前是第几次）
	round := task.RetryCount + 1
	historyErr := m.dao.AddTaskHistory(ctx, task.ID, round, historyStatus, result, startTimeUnix, endTimeUnix, int64(duration))
	if historyErr != nil {
		m.logger.Warningf(ctx, "[AsyncTask] [%s] Failed to add task history (id: %d): %v", m.getTaskTypeText(task.TaskType), task.ID, historyErr)
		// 历史记录失败不影响主流程
	}

	return nil
}

// calculateNextRetryTime 计算下次重试时间（指数退避）
func (m *AsyncTaskManager) calculateNextRetryTime(task *Task) int64 {
	retryCount := task.RetryCount
	intervals := m.config.BackoffIntervals

	var interval time.Duration
	if retryCount >= len(intervals) {
		interval = intervals[len(intervals)-1]
	} else {
		interval = intervals[retryCount]
	}

	return time.Now().Add(interval).Unix()
}

// timeoutMonitor 超时监控
func (m *AsyncTaskManager) timeoutMonitor() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.TimeoutCheckInterval)
	defer ticker.Stop()

	// 初始延迟
	time.Sleep(m.config.QueryInterval)

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			rowsAffected, err := m.dao.ResetTimeoutTasks(m.ctx, m.config.TaskTimeout)
			if err != nil {
				m.logger.Errorf(m.ctx, "[AsyncTask] Failed to reset timeout tasks: %v", err)
				continue
			}
			if rowsAffected > 0 {
				m.logger.Infof(m.ctx, "[AsyncTask] Reset %d timeout tasks", rowsAffected)
			}
		}
	}
}

// GetTaskResult 查询任务信息及执行历史
func (m *AsyncTaskManager) GetTaskResult(ctx context.Context, customID string) (map[string]interface{}, error) {
	if m.closed {
		return nil, ErrManagerClosed
	}

	// 查询任务信息
	task, err := m.dao.GetTaskByCustomID(ctx, customID)
	if err != nil {
		return nil, err
	}

	if task == nil {
		return nil, nil
	}

	// 查询执行历史
	history, err := m.dao.GetTaskHistory(ctx, task.ID)
	if err != nil {
		return nil, err
	}

	out := make(map[string]interface{})
	out["task"] = map[string]interface{}{
		"id":              task.ID,
		"custom_id":       task.CustomID,
		"task_type":       m.getTaskTypeText(task.TaskType),
		"status":          map[TaskStatus]string{TaskStatusPending: "待执行", TaskStatusProcessing: "执行中", TaskStatusSuccess: "执行成功"}[task.Status],
		"content":         task.Content,
		"retry_count":     task.RetryCount,
		"next_retry_time": task.NextRetryTime.Unix(),
		"last_error":      task.LastError,
		"version":         task.Version,
		"create_time":     task.CreateTime.Unix(),
		"update_time":     task.UpdateTime.Unix(),
	}
	his := make([]interface{}, 0, len(history))
	for _, h := range history {
		his = append(his, map[string]interface{}{
			"round":      fmt.Sprintf("第 %d 次执行", h.Round),
			"status":     map[int]string{0: "执行失败", 1: "执行成功"}[h.Status],
			"result":     h.Result,
			"start_time": h.StartTime.Format(time.DateTime),
			"end_time":   h.EndTime.Format(time.DateTime),
			"duration":   fmt.Sprintf("任务执行耗时：%4dms", h.Duration),
		})
	}
	out["history"] = his

	return out, nil
}
