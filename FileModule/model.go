package FileModule

import "time"

// FileModule 文件所属业务模块
type FileModule int

// FileType 文件类型, 使用方自定义(比如: 公司营业执照, 公司法人证件照,  logo,  banner, 海报, 文档, 视频)
type FileType int

// FileStatus 文件状态
type FileStatus int

const (
	FileStatusInit          FileStatus = iota // Init
	FileStatusUploadSuccess                   // Upload Success
	FileStatusUploadFailed                    // Upload Failed
)

func GetFileStatusText(status FileStatus) string {
	switch status {
	case FileStatusInit:
		return "Init"
	case FileStatusUploadSuccess:
		return "Upload Success"
	case FileStatusUploadFailed:
		return "Upload Failed"
	default:
		return "Unknown File Status"
	}
}

type FileInfoEntity struct {
	ID         int64  `json:"id"`
	Module     int    `json:"module"`
	CustomID   string `json:"custom_id"`
	Type       int    `json:"type"`
	FileID     string `json:"file_id"`
	FileName   string `json:"file_name"`
	FileLink   string `json:"file_link"`
	Status     int    `json:"status"`
	CreateTime int64  `json:"create_time"`
	UpdateTime int64  `json:"update_time"`
}

type FileInfo struct {
	ID       int64      `json:"id"`
	Module   FileModule `json:"module"`
	Type     FileType   `json:"type"`
	CustomID string     `json:"custom_id"`

	FileID   string     `json:"file_id"`
	FileName string     `json:"file_name"`
	FileLink string     `json:"file_link"`
	Status   FileStatus `json:"status"`

	CreateTime time.Time `json:"create_time"`
	UpdateTime time.Time `json:"update_time"`
}

func ConvertFileModel(in *FileInfoEntity) (out *FileInfo) {
	return &FileInfo{
		ID:       in.ID,
		Module:   FileModule(in.Module),
		Type:     FileType(in.Type),
		CustomID: in.CustomID,

		FileID:   in.FileID,
		FileName: in.FileName,
		FileLink: in.FileLink,
		Status:   FileStatus(in.Status),

		CreateTime: time.Unix(in.CreateTime, 0),
		UpdateTime: time.Unix(in.UpdateTime, 0),
	}
}

type PreUploadReq struct {
	FileName    string `json:"filename" dc:"文件名称"`
	ContentType string `json:"content_type" dc:"文件类型"`
	Size        int64  `json:"size" dc:"文件大小"`
	BucketID    string `json:"bucket_id" dc:"桶ID"`
}

type PreUploadRes struct {
	FileID       string `json:"file_id" dc:"文件ID"`
	OriginalName string `json:"original_name" dc:"文件原始名称"`
	FileLink     string `json:"file_link" dc:"文件链接"`
	UploadURL    string `json:"upload_url" dc:"上传URL"`
	ExpiresAt    string `json:"expires_at" dc:"过期时间"`
	ExpiresIn    int64  `json:"expires_in" dc:"过期时间"`
}

type PreDownloadRes struct {
	DownloadURL string `json:"download_url" dc:"下载URL"`
	ExpiresAt   string `json:"expires_at" dc:"过期时间"`
	ExpiresIn   int64  `json:"expires_in" dc:"过期时间"`
}
