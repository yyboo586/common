package AsyncTask

import (
	"errors"
	"fmt"
)

var (
	// ErrNoRowsAffected 并发更新错误
	ErrNoRowsAffected = errors.New("concurrent update error: no rows affected")

	// ErrHandlerNotFound 处理器未找到
	ErrHandlerNotFound = errors.New("handler not found")

	// ErrHandlerAlreadyRegistered 处理器已注册
	ErrHandlerAlreadyRegistered = errors.New("handler already registered")

	// ErrManagerClosed 管理器已关闭
	ErrManagerClosed = errors.New("manager closed")
)

// ErrInvalidConfig 无效配置错误
func ErrInvalidConfig(msg string) error {
	return fmt.Errorf("invalid config: %s", msg)
}
