package LogModule

import (
	"context"
	"time"
)

type ILogManager interface {
	// EnsureTable 确保日志表存在
	EnsureTable() error

	// Write 写入单条日志
	BatchWrite(ctx context.Context, in []*LogItem) (err error)

	// List 查询日志
	List(ctx context.Context, filter *LogListFilter) (out []*LogItem, err error)
}

// LogListFilter 日志查询条件
// 支持按模块、动作、时间范围分页查询，分页采用 Size/Offset
type LogListFilter struct {
	Module     LogModule
	Action     LogAction
	OperatorID string

	StartTime time.Time
	EndTime   time.Time

	Page int
	Size int
}
