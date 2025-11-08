package FileModule

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/os/glog"
)

var (
	fileManagerOnce     sync.Once
	fileManagerInstance *FileManager
)

// FileManager 文件管理器
type FileManager struct {
	logger *glog.Logger

	dao *fileManagerDAO

	fileEngine *fileEngine
}

// newFileManager 创建文件管理器
func NewFileManager(config *Config) (IFileManager, error) {
	fileManagerOnce.Do(func() {
		ctx := context.Background()
		daoFileManager, err := newFileManagerDAO(ctx, config)
		if err != nil {
			panic(err)
		}

		logger := glog.New()
		if config.EnableDebug {
			logger.SetLevel(glog.LEVEL_ALL)
		} else {
			logger.SetLevel(glog.LEVEL_ERRO)
		}

		logger.SetPrefix("[FileManager]")
		logger.SetTimeFormat(time.DateTime)
		logger.SetWriter(os.Stdout)

		fileManagerInstance = &FileManager{
			logger:     logger,
			dao:        daoFileManager,
			fileEngine: NewFileEngine(config),
		}
	})

	err := fileManagerInstance.EnsureTable()
	if err != nil {
		panic(err)
	}

	return fileManagerInstance, nil
}

func (m *FileManager) EnsureTable() error {
	return m.dao.EnsureTable()
}

func (m *FileManager) PreUpload(ctx context.Context, in *PreUploadReq) (out *PreUploadRes, err error) {
	out, err = m.fileEngine.PreUpload(ctx, in)
	if err != nil {
		return nil, err
	}

	err = m.dao.Create(ctx, out.FileID, out.OriginalName, out.FileLink)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (m *FileManager) PreDownload(ctx context.Context, fileID string) (out *PreDownloadRes, err error) {
	out, err = m.fileEngine.PreDownload(ctx, fileID)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (m *FileManager) UpdateStatus(ctx context.Context, fileID string, status FileStatus) (err error) {
	return m.dao.UpdateStatus(ctx, fileID, status)
}

func (m *FileManager) CreateAssociation(ctx context.Context, tx gdb.TX, fileID string, module FileModule, customID string, typ FileType) (err error) {
	return m.dao.CreateAssociation(ctx, tx, fileID, module, customID, typ)
}

func (m *FileManager) CreateAssociationBatch(ctx context.Context, tx gdb.TX, fileInfos []*FileInfo) (err error) {
	for _, fileInfo := range fileInfos {
		err = m.dao.CreateAssociation(ctx, tx, fileInfo.FileID, fileInfo.Module, fileInfo.CustomID, fileInfo.Type)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *FileManager) ClearAssociation(ctx context.Context, tx gdb.TX, module FileModule, customID string) (err error) {
	return m.dao.ClearCustomInfo(ctx, tx, module, customID)
}

func (m *FileManager) Get(ctx context.Context, fileID string) (out *FileInfo, err error) {
	return m.dao.Get(ctx, fileID)
}

func (m *FileManager) ListByModuleAndCustomID(ctx context.Context, module FileModule, customID string) (out []*FileInfo, err error) {
	return m.dao.ListByModuleAndCustomID(ctx, module, customID)
}

func (m *FileManager) ListByModuleAndCustomIDs(ctx context.Context, module FileModule, customIDs []string) (out map[string][]*FileInfo, err error) {
	return m.dao.ListByModuleAndCustomIDs(ctx, module, customIDs)
}

func (m *FileManager) IsUploadSuccess(ctx context.Context, fileInfo *FileInfo) (err error) {
	if fileInfo.Status != FileStatusUploadSuccess {
		return ErrFileUploadFailed
	}
	return nil
}

func (m *FileManager) BatchCheckUploadSuccess(ctx context.Context, fileIDs []string) (notSuccessFileIDs []string, err error) {
	if len(fileIDs) == 0 {
		return nil, nil
	}

	return m.dao.CheckFileUploadSuccess(ctx, fileIDs)
}
