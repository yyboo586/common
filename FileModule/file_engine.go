package FileModule

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/yyboo586/common/httpUtils"
)

var (
	fileEngineOnce     sync.Once
	fileEngineInstance *fileEngine
)

type fileEngine struct {
	addr   string
	client httpUtils.HTTPClient
}

func NewFileEngine(cfg *Config) *fileEngine {
	fileEngineOnce.Do(func() {
		fileEngineInstance = &fileEngine{
			addr:   cfg.FileEngineAddr,
			client: httpUtils.NewHTTPClientWithDebug(cfg.EnableDebug),
		}
	})
	return fileEngineInstance
}

func (f *fileEngine) PreUpload(ctx context.Context, in *PreUploadReq) (out *PreUploadRes, err error) {
	url := fmt.Sprintf("%s/api/v1/file-engine/files/upload-tokens", f.addr)
	fileInfo := map[string]interface{}{
		"filename":     in.FileName,
		"content_type": in.ContentType,
		"size":         in.Size,
		"bucket_id":    in.BucketID,
	}
	reqBody := map[string]interface{}{
		"file": fileInfo,
	}

	status, respBody, err := f.client.POST(ctx, url, nil, reqBody)
	if err != nil {
		return nil, gerror.Newf("pre upload file failed, err: %s", err.Error())
	}
	if status != http.StatusOK {
		return nil, gerror.Newf("pre upload file failed, status: %d, respBody: %s", status, string(respBody))
	}

	var resp map[string]interface{}
	err = json.Unmarshal(respBody, &resp)
	if err != nil {
		return nil, gerror.Newf("pre upload file failed, err: %s", err.Error())
	}

	out = &PreUploadRes{
		FileID:       resp["id"].(string),
		OriginalName: resp["original_name"].(string),
		FileLink:     resp["visit_url"].(string),
		UploadURL:    resp["upload_url"].(string),
		ExpiresAt:    resp["expires_at"].(string),
		ExpiresIn:    int64(resp["expires_in"].(float64)),
	}
	return
}

func (f *fileEngine) PreDownload(ctx context.Context, fileID string) (out *PreDownloadRes, err error) {
	url := fmt.Sprintf("%s/api/v1/file-engine/files/%s/download-tokens", f.addr, fileID)

	status, respBody, err := f.client.GET(ctx, url, nil)
	if err != nil {
		return nil, gerror.Newf("pre download file failed, err: %s", err.Error())
	}
	if status != http.StatusOK {
		return nil, gerror.Newf("pre download file failed, status: %d, respBody: %s", status, string(respBody))
	}

	var resp map[string]interface{}
	err = json.Unmarshal(respBody, &resp)
	if err != nil {
		return nil, gerror.Newf("pre download file failed, err: %s", err.Error())
	}

	out = &PreDownloadRes{
		DownloadURL: resp["download_url"].(string),
		ExpiresAt:   resp["expires_at"].(string),
		ExpiresIn:   int64(resp["expires_in"].(float64)),
	}
	return out, nil
}

func (f *fileEngine) Delete(ctx context.Context, fileID string) (err error) {
	url := fmt.Sprintf("%s/api/v1/file-engine/files/%s", f.addr, fileID)

	status, respBody, err := f.client.DELETE(ctx, url, nil, nil)
	if err != nil {
		return gerror.Newf("delete file failed, err: %s", err.Error())
	}
	if status != http.StatusOK {
		return gerror.Newf("delete file failed, status: %d, respBody: %s", status, string(respBody))
	}

	return nil
}

func (f *fileEngine) ReportUploadResult(ctx context.Context, fileID string, success bool) (err error) {
	url := fmt.Sprintf("%s/api/v1/file-engine/files/%s/status", f.addr, fileID)

	reqBody := map[string]interface{}{
		"success": success,
	}

	status, respBody, err := f.client.POST(ctx, url, nil, reqBody)
	if err != nil {
		return gerror.Newf("report upload result failed, err: %s", err.Error())
	}
	if status != http.StatusOK {
		return gerror.Newf("report upload result failed, status: %d, respBody: %s", status, string(respBody))
	}

	return nil
}
