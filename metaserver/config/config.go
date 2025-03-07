package config

type ServerConfig struct {
	HTTPPort          int          `yaml:"http_port" validate:"required,min=1024"`           // 服务监听端口
	DBConnection      string       `yaml:"db_connection" validate:"required"`                // 数据库连接字符串
	HeartbeatInterval int          `yaml:"heartbeat_interval" validate:"min=5"`              // 心跳检测间隔（秒）
	StorageNodes      []NodeConfig `yaml:"storage_nodes" validate:"dive"`                    // 初始存储节点列表
	MaxConnections    int          `yaml:"max_connections" validate:"min=1"`                 // 最大连接数
	LogLevel          string       `yaml:"log_level" validate:"oneof=debug info warn error"` // 日志级别
	Timeout           int          `yaml:"timeout" validate:"min=1"`                         // 请求超时时间（秒）
}

type NodeConfig struct {
	ID       string `yaml:"id" validate:"required"`        // 节点唯一标识
	Address  string `yaml:"address" validate:"ipv4"`       // 节点IP地址
	Capacity int64  `yaml:"capacity" validate:"min=1024"`  // 存储容量（MB）
	IsActive bool   `yaml:"is_active" validate:"required"` // 节点是否处于激活状态
	Region   string `yaml:"region" validate:"required"`    // 节点所在区域
}

// 提供默认配置生成方法
func DefaultConfig() *ServerConfig {
	return &ServerConfig{
		HTTPPort:          8080,
		HeartbeatInterval: 15,
		MaxConnections:    100,
		LogLevel:          "info",
		Timeout:           30,
	}
}
