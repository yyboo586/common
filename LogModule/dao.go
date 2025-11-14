package LogModule

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

// LogManagerDAO 封装日志模块的数据库访问逻辑
// 提供表初始化、批量写入、查询能力
type LogManagerDAO struct {
	group     string
	tableName string
	db        gdb.DB
	maxBatch  int
	ctx       context.Context
}

func newLogManagerDAO(ctx context.Context, config *Config) (*LogManagerDAO, error) {
	if config.Group == "" {
		config.Group = "default"
	}
	if config.TableName == "" {
		config.TableName = "t_log"
	}
	if config.MaxBatch <= 0 {
		config.MaxBatch = 200
	}

	gdb.SetConfig(gdb.Config{
		config.Group: gdb.ConfigGroup{
			{Link: config.DSN},
		},
	})

	db := g.DB(config.Group)
	if db == nil {
		return nil, gerror.Newf("failed to get database instance for group: %s", config.Group)
	}

	if config.EnableDebug {
		db.SetDebug(true)
	}

	return &LogManagerDAO{
		group:     config.Group,
		tableName: config.TableName,
		db:        db,
		maxBatch:  config.MaxBatch,
		ctx:       ctx,
	}, nil
}

func (d *LogManagerDAO) EnsureTable() error {
	createTableSQL := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
    id BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    module TINYINT(1) NOT NULL DEFAULT 0 COMMENT '业务模块',
    action TINYINT(1) NOT NULL DEFAULT 0 COMMENT '业务动作',
    message VARCHAR(255) NOT NULL DEFAULT '' COMMENT '日志概要',
    detail TEXT COMMENT '日志详情(JSON)',
    operator_id VARCHAR(40) NOT NULL DEFAULT '' COMMENT '操作人ID',
    ip VARCHAR(64) NOT NULL DEFAULT '' COMMENT '操作人IP',
    create_time BIGINT(20) NOT NULL COMMENT '创建时间(秒)',
    PRIMARY KEY (id),
    KEY idx_module_action_time (module, action, operator_id, create_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通用日志表';`, d.tableName)

	_, err := d.db.Exec(d.ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create log table: %w", err)
	}
	return nil
}

func (d *LogManagerDAO) BatchCreate(ctx context.Context, entities []*LogEntity) error {
	if len(entities) == 0 {
		return nil
	}

	size := d.maxBatch
	if size <= 0 {
		size = len(entities)
	}

	for start := 0; start < len(entities); start += size {
		end := start + size
		if end > len(entities) {
			end = len(entities)
		}

		chunk := entities[start:end]
		data := make([]g.Map, 0, len(chunk))
		for _, entity := range chunk {
			if entity == nil {
				continue
			}

			data = append(data, g.Map{
				"module":      entity.Module,
				"action":      entity.Action,
				"message":     entity.Message,
				"detail":      entity.Detail,
				"operator_id": entity.OperatorID,
				"ip":          entity.IP,
				"create_time": entity.CreateTime,
			})
		}

		if len(data) == 0 {
			continue
		}

		if _, err := d.db.Model(d.tableName).Ctx(ctx).Data(data).Insert(); err != nil {
			return err
		}
	}

	return nil
}

func (d *LogManagerDAO) List(ctx context.Context, filter *LogListFilter) ([]*LogEntity, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Size <= 0 {
		filter.Size = 10
	}

	model := d.buildFilterModel(ctx, filter)
	model = model.Page(filter.Page, filter.Size).OrderDesc("create_time")

	var entities []*LogEntity
	if err := model.Scan(&entities); err != nil {
		return nil, err
	}

	return entities, nil
}

func (d *LogManagerDAO) buildFilterModel(ctx context.Context, filter *LogListFilter) *gdb.Model {
	model := d.db.Model(d.tableName).Ctx(ctx)
	if filter == nil {
		return model
	}

	if filter.Module != 0 {
		model = model.Where("module = ?", int(filter.Module))
	}
	if filter.Action != 0 {
		model = model.Where("action = ?", int(filter.Action))
	}
	if filter.OperatorID != "" {
		model = model.Where("operator_id = ?", filter.OperatorID)
	}
	if !filter.StartTime.IsZero() {
		model = model.Where("create_time >= ?", filter.StartTime.Unix())
	}
	if !filter.EndTime.IsZero() {
		model = model.Where("create_time <= ?", filter.EndTime.Unix())
	}

	return model
}
