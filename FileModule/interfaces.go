package FileModule

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
)

type IFileManager interface {
	// 获取文件上传链接
	PreUpload(ctx context.Context, in *PreUploadReq) (out *PreUploadRes, err error)
	// 获取文件下载链接
	PreDownload(ctx context.Context, fileID string) (out *PreDownloadRes, err error)

	// 更新文件状态
	UpdateStatus(ctx context.Context, fileID string, status FileStatus) (err error)
	// 创建文件关联（部分场景下，预上传文件时，不会将文件与业务关联，需要后续创建关联）
	CreateAssociation(ctx context.Context, tx gdb.TX, fileID string, module FileModule, customID string, typ FileType) (err error)
	// 批量创建文件关联
	CreateAssociationBatch(ctx context.Context, tx gdb.TX, fileInfos []*FileInfo) (err error)
	// 清除文件 自定义属性
	ClearAssociation(ctx context.Context, tx gdb.TX, module FileModule, customID string) (err error)

	// 获取文件
	Get(ctx context.Context, fileID string) (out *FileInfo, err error)
	// 按模块与自定义ID获取文件列表
	ListByModuleAndCustomID(ctx context.Context, module FileModule, customID string) (out []*FileInfo, err error)
	// 按模块与自定义IDs获取文件列表
	ListByModuleAndCustomIDs(ctx context.Context, module FileModule, customIDs []string) (out map[string][]*FileInfo, err error)

	// 检查文件是否上传成功
	IsUploadSuccess(ctx context.Context, fileInfo *FileInfo) (err error)
	// 批量检查文件是否上传成功
	BatchCheckUploadSuccess(ctx context.Context, fileIDs []string) (notSuccessFileIDs []string, err error)
}
