package AsyncTask

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
)

// TaskType 任务类型（使用方自定义）
type TaskType int

// TaskStatus 任务状态
type TaskStatus int

const (
	TaskStatusPending    TaskStatus = iota // 待执行
	TaskStatusProcessing                   // 执行中
	TaskStatusSuccess                      // 执行成功
)

// TaskHandler 任务处理函数
type TaskHandler func(ctx context.Context, task *Task) error

// TaskEntity 数据库实体
type TaskEntity struct {
	ID            int64  `orm:"id"`
	CustomID      string `orm:"custom_id"`
	TaskType      int    `orm:"task_type"`
	Status        int    `orm:"status"`
	Content       string `orm:"content"`
	RetryCount    int    `orm:"retry_count"`
	NextRetryTime int64  `orm:"next_retry_time"`
	LastError     string `orm:"last_error"`
	Version       int    `orm:"version"`
	CreateTime    int64  `orm:"create_time"`
	UpdateTime    int64  `orm:"update_time"`
}

// Task 任务模型
type Task struct {
	ID            int64       `json:"id"`
	CustomID      string      `json:"custom_id"`
	TaskType      TaskType    `json:"task_type"`
	Status        TaskStatus  `json:"status"`
	Content       interface{} `json:"content"`
	RetryCount    int         `json:"retry_count"`
	NextRetryTime time.Time   `json:"next_retry_time"`
	LastError     string      `json:"last_error"`
	Version       int         `json:"version"`
	CreateTime    time.Time   `json:"create_time"`
	UpdateTime    time.Time   `json:"update_time"`
}

// TaskHistory 任务执行历史
type TaskHistory struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	Round     int       `json:"round"`
	Status    int       `json:"status"` // 0:失败, 1:成功
	Result    string    `json:"result"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  int64     `json:"duration"` // 执行时长（秒）
}

// TaskHistoryEntity 任务执行历史数据库实体
type TaskHistoryEntity struct {
	ID        int64  `orm:"id"`
	TaskID    int64  `orm:"task_id"`
	Round     int    `orm:"round"`
	Status    int    `orm:"status"`
	Result    string `orm:"result"`
	StartTime int64  `orm:"start_time"`
	EndTime   int64  `orm:"end_time"`
	Duration  int64  `orm:"duration"`
}

func ConvertTaskEntityToTask(in *TaskEntity) (out *Task, err error) {
	var contentData interface{}
	err = json.Unmarshal([]byte(in.Content), &contentData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal content: %w", err)
	}

	out = &Task{
		ID:            in.ID,
		CustomID:      in.CustomID,
		TaskType:      TaskType(in.TaskType),
		Status:        TaskStatus(in.Status),
		Content:       contentData,
		RetryCount:    in.RetryCount,
		NextRetryTime: time.Unix(in.NextRetryTime, 0),
		LastError:     in.LastError,
		Version:       in.Version,
		CreateTime:    time.Unix(in.CreateTime, 0),
		UpdateTime:    time.Unix(in.UpdateTime, 0),
	}
	return out, nil
}

func ConvertTaskHistoryEntityToTaskHistory(in *TaskHistoryEntity) (out *TaskHistory) {
	return &TaskHistory{
		ID:        in.ID,
		TaskID:    in.TaskID,
		Round:     in.Round,
		Status:    in.Status,
		Result:    in.Result,
		StartTime: time.Unix(in.StartTime, 0),
		EndTime:   time.Unix(in.EndTime, 0),
		Duration:  in.Duration,
	}
}

type Manager interface {
	// 添加即时任务（支持事务）
	AddTask(ctx context.Context, tx gdb.TX, taskType TaskType, customID string, content []byte) error
	// 添加定时任务（支持事务）
	AddScheduledTask(ctx context.Context, tx gdb.TX, taskType TaskType, customID string, content []byte, scheduledTime time.Time) error

	// 注册任务处理器
	RegisterHandler(taskType TaskType, taskTypeText string, handler TaskHandler) error
	// 启动异步任务处理
	Start() error
	// 停止异步任务处理
	Stop()
	// 唤醒任务处理线程
	WakeUp(taskType TaskType)

	// 查询任务信息及执行历史
	GetTaskResult(ctx context.Context, customID string) (map[string]interface{}, error)
	// 查询任务是否已存在
	IsTaskExists(ctx context.Context, customID string, taskType TaskType) (bool, error)
}
