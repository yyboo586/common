package LogModule

import (
	"context"

	"github.com/gogf/gf/v2/errors/gerror"
)

// LogManager 日志模块核心实现
// 负责初始化 DAO、提供批量写入与查询能力
type LogManager struct {
	dao *LogManagerDAO
}

// NewLogManager 创建日志管理器实例
// config 允许为空，为空时使用 DefaultConfig
func NewLogManager(config *Config) (ILogManager, error) {
	if config == nil {
		config = DefaultConfig()
	}
	if config.DSN == "" {
		return nil, gerror.New("LogModule config DSN is required")
	}

	dao, err := newLogManagerDAO(context.Background(), config)
	if err != nil {
		return nil, err
	}

	manager := &LogManager{dao: dao}
	if err = manager.EnsureTable(); err != nil {
		return nil, err
	}

	return manager, nil
}

func (m *LogManager) EnsureTable() error {
	return m.dao.EnsureTable()
}

func (m *LogManager) BatchWrite(ctx context.Context, in []*LogItem) error {
	if len(in) == 0 {
		return nil
	}

	entities := make([]*LogEntity, 0, len(in))
	for _, item := range in {
		entity := NewLogEntityFromItem(item)
		if entity != nil {
			entities = append(entities, entity)
		}
	}

	if len(entities) == 0 {
		return nil
	}

	return m.dao.BatchCreate(ctx, entities)
}

func (m *LogManager) List(ctx context.Context, filter *LogListFilter) ([]*LogItem, error) {
	entities, err := m.dao.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	items := make([]*LogItem, 0, len(entities))
	for _, entity := range entities {
		if item := ConvertLogItem(entity); item != nil {
			items = append(items, item)
		}
	}

	return items, nil
}
