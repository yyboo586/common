package FileModule

// Config FileModule配置
type Config struct {
	// 数据库DSN，格式: mysql:user:password@tcp(host:port)/database?parseTime=true
	DSN string

	// 数据库名称
	Database string

	// 数据库分组名，默认为"default"
	Group string

	// 文件引擎服务地址
	FileEngineAddr string

	// 是否开启调试模式
	EnableDebug bool
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Group: "default",
	}
}
