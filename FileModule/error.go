package FileModule

import "github.com/gogf/gf/v2/errors/gerror"

var (
	ErrFileNotFound     = gerror.New("文件不存在")
	ErrFileUploadFailed = gerror.New("文件上传失败")
)
