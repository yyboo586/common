package LogModule

// Config LogModule 配置信息
// 参考 FileModule 的配置结构，主要用于数据库初始化与日志输出控制
//
// 字段说明：
//   DSN         数据库连接串，支持 mysql/postgres 等 gdb 支持的驱动
//   Group       gdb 分组名称，用于多库场景隔离，默认 "log"
//   TableName   日志表名称，默认 "t_log"
//   EnableDebug 是否开启 gf 数据库调试与日志器调试
//   MaxBatch    BatchWrite 单次写库的最大行数，超过后自动拆分
//   LogLevel    组件内部使用的日志级别（debug/info/warn/error）
//
// 业务可以基于该配置扩展，如配置分库分表策略、外部日志服务地址等

type Config struct {
	DSN         string
	Group       string
	TableName   string
	EnableDebug bool
	MaxBatch    int
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Group:       "default",
		TableName:   "t_log",
		EnableDebug: true,
		MaxBatch:    200,
	}
}
