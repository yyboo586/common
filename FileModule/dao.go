package FileModule

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

// fileManagerDAO 数据访问对象
type fileManagerDAO struct {
	group     string
	tableName string
	db        gdb.DB
	ctx       context.Context
}

// newDAO 创建DAO实例
func newFileManagerDAO(ctx context.Context, config *Config) (*fileManagerDAO, error) {
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

	if config.EnableDebug {
		db.SetDebug(true)
	}

	dao := &fileManagerDAO{
		group:     config.Group,
		tableName: "t_file",
		db:        db,
		ctx:       ctx,
	}

	return dao, nil
}

// EnsureTable 确保表存在，不存在则创建
func (d *fileManagerDAO) EnsureTable() error {
	// 创建文件表
	createTableSQL :=
		`
CREATE TABLE IF NOT EXISTS t_file (
    id BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    module TINYINT(1) DEFAULT 0 COMMENT '业务模块',
    custom_id VARCHAR(40) DEFAULT '' COMMENT '业务自定义ID',
    type TINYINT(1) DEFAULT 0 COMMENT '文件类型',

    file_id VARCHAR(40) NOT NULL COMMENT '文件ID',
    file_orininal_name VARCHAR(255) NOT NULL COMMENT '文件原始名称',
    file_link TEXT COMMENT '文件链接',
    status TINYINT(1) DEFAULT 0 COMMENT '文件状态(0:初始化,1:上传成功,2:上传失败)',
    
	create_time BIGINT(20) NOT NULL COMMENT '创建时间',
    update_time BIGINT(20) COMMENT '更新时间',
    PRIMARY KEY (id),
    KEY idx_module_id_type (module, custom_id, type),
    UNIQUE KEY idx_file_id_status (file_id, status)
) ENGINE=InnoDB COMMENT='文件信息表';
`

	_, err := d.db.Exec(d.ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

func (d *fileManagerDAO) Columns() string {
	return "id, module, custom_id, type, file_id, file_orininal_name, file_link, status, create_time, update_time"
}

func (d *fileManagerDAO) Create(ctx context.Context, fileID string, fileName string, fileLink string) (err error) {
	dataInsert := g.Map{
		"file_id":            fileID,
		"file_orininal_name": fileName,
		"file_link":          fileLink,
		"status":             FileStatusInit,
		"create_time":        time.Now().Unix(),
		"update_time":        time.Now().Unix(),
	}

	_, err = d.db.Model(d.tableName).Ctx(ctx).Data(dataInsert).Insert()
	if err != nil {
		return err
	}

	return nil
}

func (d *fileManagerDAO) UpdateStatus(ctx context.Context, fileID string, status FileStatus) (err error) {
	dataUpdate := g.Map{
		"status":      status,
		"update_time": time.Now().Unix(),
	}

	result, err := d.db.Model(d.tableName).Ctx(ctx).Data(dataUpdate).Where("file_id = ?", fileID).Update()
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrFileNotFound
	}

	return nil
}

func (d *fileManagerDAO) CreateAssociation(ctx context.Context, tx gdb.TX, fileID string, module FileModule, customID string, typ FileType) (err error) {
	dataUpdate := g.Map{
		"module":      module,
		"custom_id":   customID,
		"type":        typ,
		"update_time": time.Now().Unix(),
	}

	result, err := d.db.Model(d.tableName).Ctx(ctx).TX(tx).Data(dataUpdate).Where("file_id = ?", fileID).Update()
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrFileNotFound
	}

	return nil
}

func (d *fileManagerDAO) Get(ctx context.Context, fileID string) (out *FileInfo, err error) {
	var entity FileInfoEntity

	err = d.db.Model(d.tableName).Ctx(ctx).Where("file_id = ?", fileID).Scan(&entity)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	return ConvertFileModel(&entity), nil
}

func (d *fileManagerDAO) ListByModuleAndCustomID(ctx context.Context, module FileModule, customID string) (out []*FileInfo, err error) {
	var entities []FileInfoEntity

	err = d.db.Model(d.tableName).Ctx(ctx).Where("module = ?", module).Where("custom_id = ?", customID).Scan(&entities)
	if err != nil {
		return nil, err
	}

	out = make([]*FileInfo, 0, len(entities))
	for _, entity := range entities {
		out = append(out, ConvertFileModel(&entity))
	}
	return out, nil
}

func (d *fileManagerDAO) ListByModuleAndCustomIDs(ctx context.Context, module FileModule, customIDs []string) (out map[string][]*FileInfo, err error) {
	var entities []FileInfoEntity

	err = d.db.Model(d.tableName).Ctx(ctx).Where("module = ?", module).Where("custom_id IN (?)", customIDs).Scan(&entities)
	if err != nil {
		return nil, err
	}

	out = make(map[string][]*FileInfo, len(entities))
	for _, entity := range entities {
		out[entity.CustomID] = append(out[entity.CustomID], ConvertFileModel(&entity))
	}
	return out, nil
}

func (d *fileManagerDAO) ClearCustomInfo(ctx context.Context, tx gdb.TX, module FileModule, customID string) (err error) {
	dataUpdate := g.Map{
		"module":      0,
		"custom_id":   "",
		"type":        0,
		"update_time": time.Now().Unix(),
	}

	result, err := d.db.Model(d.tableName).Ctx(ctx).TX(tx).
		Data(dataUpdate).
		Where("module = ?", module).
		Where("custom_id = ?", customID).
		Update()
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrFileNotFound
	}

	return nil
}

func (d *fileManagerDAO) CheckFileUploadSuccess(ctx context.Context, fileIDs []string) (notSuccessFileIDs []string, err error) {
	if len(fileIDs) == 0 {
		return nil, nil
	}

	// 使用一次SQL查询检查所有文件是否都上传成功
	// 查询状态为UploadSuccess的文件数量
	successCount, err := d.db.Model(d.tableName).Ctx(ctx).
		Where("file_id IN (?)", fileIDs).
		Where("status = ?", FileStatusUploadSuccess).
		Count()
	if err != nil {
		return nil, err
	}

	// 如果成功上传的文件数量不等于传入的fileIDs数量，说明有问题
	if successCount != len(fileIDs) {
		var notSuccessRecords []gdb.Record
		// 检查是否有文件不存在
		err := d.db.Model(d.tableName).Ctx(ctx).
			Fields("file_id").
			Where("file_id IN (?)", fileIDs).
			Where("status != ?", FileStatusUploadSuccess).
			Scan(&notSuccessRecords)
		if err != nil {
			return nil, err
		}

		for _, record := range notSuccessRecords {
			notSuccessFileIDs = append(notSuccessFileIDs, record["file_id"].String())
		}
	}

	return notSuccessFileIDs, nil
}
