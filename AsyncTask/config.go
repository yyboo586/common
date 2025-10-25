package AsyncTask

import (
	"time"

	"github.com/gogf/gf/v2/os/glog"
)

// Config AsyncTask配置
type Config struct {
	// 数据库DSN，格式: mysql:user:password@tcp(host:port)/database?parseTime=true
	DSN string

	// 数据库名称
	Database string

	// 数据库分组名，默认为"default"
	Group string

	// 表名，默认为"t_async_task"
	TableName string

	// 历史表名，默认为"t_async_task_history"
	HistoryTableName string

	// 日志配置
	LogLevel int

	// 工作线程初始化间隔，默认10秒
	InitInterval time.Duration

	// 工作线程查询间隔（无任务时），默认30秒
	QueryInterval time.Duration

	// 错误休眠间隔，默认3秒
	ErrSleepInterval time.Duration

	// 退避重试间隔列表
	BackoffIntervals []time.Duration

	// 超时监控间隔，默认24小时
	TimeoutCheckInterval time.Duration

	// 任务超时时长，默认24小时
	TaskTimeout time.Duration
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Group:                "default",
		TableName:            "t_async_task",
		HistoryTableName:     "t_async_task_history",
		LogLevel:             glog.LEVEL_INFO,
		InitInterval:         10 * time.Second,
		QueryInterval:        30 * time.Second,
		ErrSleepInterval:     3 * time.Second,
		TimeoutCheckInterval: 24 * time.Hour,
		TaskTimeout:          24 * time.Hour,
		BackoffIntervals: []time.Duration{
			2 * time.Second,
			3 * time.Second,
			5 * time.Second,
			10 * time.Second,
			30 * time.Second,
			1 * time.Minute,
			5 * time.Minute,
		},
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.DSN == "" {
		return ErrInvalidConfig("DSN is required")
	}
	if c.Database == "" {
		return ErrInvalidConfig("Database is required")
	}
	if c.Group == "" {
		c.Group = "default"
	}
	if c.TableName == "" {
		c.TableName = "t_async_task"
	}
	if c.HistoryTableName == "" {
		c.HistoryTableName = "t_async_task_history"
	}
	if c.LogLevel == 0 {
		c.LogLevel = glog.LEVEL_INFO
	}
	if c.InitInterval == 0 {
		c.InitInterval = 10 * time.Second
	}
	if c.QueryInterval == 0 {
		c.QueryInterval = 30 * time.Second
	}
	if c.ErrSleepInterval == 0 {
		c.ErrSleepInterval = 3 * time.Second
	}
	if c.TimeoutCheckInterval == 0 {
		c.TimeoutCheckInterval = 24 * time.Hour
	}
	if c.TaskTimeout == 0 {
		c.TaskTimeout = 24 * time.Hour
	}
	if len(c.BackoffIntervals) == 0 {
		c.BackoffIntervals = []time.Duration{
			2 * time.Second,
			3 * time.Second,
			5 * time.Second,
			10 * time.Second,
			30 * time.Second,
			1 * time.Minute,
			5 * time.Minute,
		}
	}
	return nil
}
