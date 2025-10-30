package AsyncTask

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/glog"
	"github.com/gogf/gf/v2/os/gtime"
)

// DAO 数据访问对象
type DAO struct {
	group            string
	tableName        string
	historyTableName string
	db               gdb.DB
	logger           *glog.Logger
	ctx              context.Context
}

// newDAO 创建DAO实例
func newDAO(ctx context.Context, config *Config) (*DAO, error) {
	// 设置数据库配置
	gdb.SetConfig(gdb.Config{
		config.Group: gdb.ConfigGroup{
			gdb.ConfigNode{
				Link: config.DSN,
			},
		},
	})

	db := g.DB(config.Group)
	if db == nil {
		return nil, gerror.Newf("failed to get database instance for group: %s", config.Group)
	}

	db.SetDebug(true)

	dao := &DAO{
		group:            config.Group,
		tableName:        config.TableName,
		historyTableName: config.HistoryTableName,
		db:               db,
		logger:           config.Logger,
		ctx:              ctx,
	}

	return dao, nil
}

// EnsureTable 确保表存在，不存在则创建
func (d *DAO) EnsureTable() error {
	// 创建任务表
	createTableSQL := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
  id BIGINT(20) AUTO_INCREMENT NOT NULL COMMENT '主键ID',
  custom_id VARCHAR(40) DEFAULT '' COMMENT '自定义任务ID',
  task_type TINYINT(1) NOT NULL COMMENT '任务类型',
  status TINYINT(1) NOT NULL DEFAULT 0 COMMENT '任务状态(0:Pending, 1:Processing, 2:Success)',
  content TEXT NOT NULL COMMENT '任务内容',
  retry_count INT(11) NOT NULL DEFAULT 0 COMMENT '重试次数',  
  next_retry_time BIGINT(20) NOT NULL COMMENT '下次处理时间(默认等于创建时间)',
  last_error TEXT COMMENT '上次任务执行失败的原因',
  version INT(11) NOT NULL DEFAULT 0 COMMENT '版本标识',  
  create_time BIGINT(20) NOT NULL COMMENT '创建时间',
  update_time BIGINT(20) NOT NULL COMMENT '更新时间',
  PRIMARY KEY (id),
  KEY idx_custom_id (custom_id),
  KEY idx_type_status_time (task_type, status, next_retry_time),
  KEY idx_status_next_retry_time (status, next_retry_time),
  KEY idx_status_update_time (status, update_time)
) ENGINE=InnoDB COMMENT='异步任务表'
`, d.tableName)

	_, err := d.db.Exec(d.ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	d.logger.Info(d.ctx, "[AsyncTask] Table %s ensured", d.tableName)

	// 创建历史表
	createHistoryTableSQL := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
    id BIGINT(20) AUTO_INCREMENT NOT NULL COMMENT '主键ID',
    task_id BIGINT(20) NOT NULL COMMENT '任务ID',
    round INT(11) NOT NULL COMMENT '第几次执行',
    status TINYINT(1) NOT NULL DEFAULT 0 COMMENT '任务是否执行成功(0:失败, 1:成功)',
    result TEXT COMMENT '任务执行结果',
    start_time BIGINT(20) NOT NULL COMMENT '任务执行开始时间',
    end_time BIGINT(20) NOT NULL COMMENT '任务执行结束时间',
    duration BIGINT(20) NOT NULL COMMENT '任务执行时间间隔',
    PRIMARY KEY (id),
    KEY idx_task_id (task_id)
) ENGINE=InnoDB COMMENT='任务执行记录表'
`, d.historyTableName)

	_, err = d.db.Exec(d.ctx, createHistoryTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create history table: %w", err)
	}

	d.logger.Info(d.ctx, "[AsyncTask] History table %s ensured", d.historyTableName)
	return nil
}

// AddTask 添加即时任务
func (d *DAO) AddTask(ctx context.Context, tx gdb.TX, taskType TaskType, customID string, content []byte) (err error) {
	data := g.Map{
		"custom_id":       customID,
		"task_type":       int(taskType),
		"content":         string(content),
		"next_retry_time": gtime.Now().Unix(),
		"create_time":     gtime.Now().Unix(),
		"update_time":     gtime.Now().Unix(),
	}

	_, err = d.db.Model(d.tableName).Ctx(ctx).TX(tx).Insert(data)
	return
}

// AddScheduledTask 添加定时任务
func (d *DAO) AddScheduledTask(ctx context.Context, tx gdb.TX, taskType TaskType, customID string, content []byte, scheduledTime time.Time) (err error) {
	data := g.Map{
		"custom_id":       customID,
		"task_type":       int(taskType),
		"content":         string(content),
		"next_retry_time": scheduledTime.Unix(),
		"create_time":     gtime.Now().Unix(),
		"update_time":     gtime.Now().Unix(),
	}

	_, err = d.db.Model(d.tableName).Ctx(ctx).TX(tx).Insert(data)
	return
}

// FetchPendingTask 获取待处理任务（乐观锁）
func (d *DAO) FetchPendingTask(ctx context.Context, taskType TaskType) (out *Task, err error) {
	var entity TaskEntity

	// 查询待处理任务
	err = d.db.Model(d.tableName).Ctx(ctx).
		Where("task_type", int(taskType)).
		Where("status", int(TaskStatusPending)).
		WhereLT("next_retry_time", gtime.Now().Unix()).
		OrderAsc("next_retry_time").
		Scan(&entity)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// 尝试更新状态为处理中（乐观锁）
	result, err := d.db.Model(d.tableName).Ctx(ctx).
		Where("id", entity.ID).
		Where("version", entity.Version).
		Data(g.Map{
			"status":      int(TaskStatusProcessing),
			"version":     entity.Version + 1,
			"update_time": gtime.Now().Unix(),
		}).
		Update()
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	if rowsAffected == 0 {
		return nil, ErrNoRowsAffected
	}

	entity.Version = entity.Version + 1
	out, err = ConvertTaskEntityToTask(&entity)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetMinNextRetryTime 获取下次执行时间最小的任务
func (d *DAO) GetMinNextRetryTime(ctx context.Context, taskType TaskType) (out *Task, err error) {
	var entity TaskEntity

	err = d.db.Model(d.tableName).Ctx(ctx).
		Where("task_type", int(taskType)).
		Where("status", int(TaskStatusPending)).
		OrderAsc("next_retry_time").
		Limit(1).
		Scan(&entity)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	out, err = ConvertTaskEntityToTask(&entity)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateTaskStatus 更新任务状态（乐观锁）
func (d *DAO) UpdateTaskStatus(ctx context.Context, task *Task, status TaskStatus, nextRetryTime int64, lastError string) error {
	data := g.Map{
		"status":          int(status),
		"version":         task.Version + 1,
		"next_retry_time": nextRetryTime,
		"update_time":     gtime.Now().Unix(),
		"last_error":      lastError,
	}

	if status == TaskStatusPending {
		data["retry_count"] = task.RetryCount + 1
	}

	result, err := d.db.Model(d.tableName).Ctx(ctx).
		Where("id", task.ID).
		Where("version", task.Version).
		Data(data).
		Update()
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNoRowsAffected
	}

	return nil
}

// ResetTimeoutTasks 重置超时任务
func (d *DAO) ResetTimeoutTasks(ctx context.Context, timeout time.Duration) (rowsAffected int64, err error) {
	timeoutTimestamp := gtime.Now().Add(-timeout).Unix()

	result, err := d.db.Model(d.tableName).Ctx(ctx).
		Where("status", int(TaskStatusProcessing)).
		WhereLT("update_time", timeoutTimestamp).
		Data(g.Map{
			"status":      int(TaskStatusPending),
			"version":     gdb.Raw("version + 1"),
			"update_time": gtime.Now().Unix(),
		}).
		Update()
	if err != nil {
		return 0, err
	}
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

// AddTaskHistory 添加任务执行历史记录
func (d *DAO) AddTaskHistory(ctx context.Context, taskID int64, round int, status int, result string, startTime, endTime int64, duration int64) error {
	data := g.Map{
		"task_id":    taskID,
		"round":      round,
		"status":     status,
		"result":     result,
		"start_time": startTime,
		"end_time":   endTime,
		"duration":   duration,
	}

	_, err := d.db.Model(d.historyTableName).Ctx(ctx).Insert(data)
	return err
}

// GetTaskByCustomID 根据 custom_id 查询任务
func (d *DAO) GetTaskByCustomID(ctx context.Context, customID string) (out *Task, err error) {
	var entity TaskEntity

	err = d.db.Model(d.tableName).Ctx(ctx).
		Where("custom_id", customID).
		Scan(&entity)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	out, err = ConvertTaskEntityToTask(&entity)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetTaskHistory 获取任务执行历史
func (d *DAO) GetTaskHistory(ctx context.Context, taskID int64) (out []*TaskHistory, err error) {
	var entities []TaskHistoryEntity

	err = d.db.Model(d.historyTableName).Ctx(ctx).
		Where("task_id", taskID).
		OrderAsc("round").
		Scan(&entities)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*TaskHistory{}, nil
		}
		return nil, err
	}

	out = make([]*TaskHistory, 0, len(entities))
	for _, entity := range entities {
		out = append(out, ConvertTaskHistoryEntityToTaskHistory(&entity))
	}

	return out, nil
}
