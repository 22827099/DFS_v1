package config

import (
	"encoding/json"
	"os"
	"time"
)

// Config 保存元数据服务器的所有配置
type Config struct {
	Server          ServerConfig   `json:"server"`      // 服务器配置
	Database        DatabaseConfig `json:"database"`    // 数据库配置
	Cluster         ClusterConfig  `json:"cluster"`     // 集群配置
	Security        SecurityConfig `json:"security"`    // 安全配置
	Logging         LoggingConfig  `json:"logging"`     // 日志配置
	ShutdownTimeout time.Duration  `json:"-"`           // 不从JSON加载，使用默认值
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Address      string        `json:"address"`
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	EnableCORS   bool     	   `json:"enable_cors"`    // 是否启用CORS
    AllowOrigins []string 	   `json:"allow_origins"`  // 允许的来源
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type            string `json:"type"`              // 数据库类型: sqlite, mysql, postgres 等
	Host            string `json:"host"`              // 数据库服务器主机地址
	Port            int    `json:"port"`              // 数据库服务器端口
	User            string `json:"user"`              // 数据库用户名
	Password        string `json:"password"`          // 数据库密码
	Database        string `json:"database"`          // 数据库名称
	MaxOpenConns    int    `json:"max_open_conns"`    // 最大打开连接数
	MaxIdleConns    int    `json:"max_idle_conns"`    // 最大空闲连接数
	ConnMaxLifetime int    `json:"conn_max_lifetime"` // 连接最大生存时间(秒)
	DSN             string `json:"dsn"`               // 数据源名称(连接字符串)
	MaxConns        int    `json:"max_connections"`   // 最大连接数
}

// ClusterConfig 集群配置
type ClusterConfig struct {
	// 节点标识
	NodeID string `json:"node_id"`
	
	// 集群成员配置
	Peers []string `json:"peers"`
	
	// 选举配置
	ElectionTimeout  time.Duration `json:"election_timeout"`
	HeartbeatTimeout time.Duration `json:"heartbeat_timeout"`
	
	// 心跳配置
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	SuspectTimeout    time.Duration `json:"suspect_timeout"`
	DeadTimeout       time.Duration `json:"dead_timeout"`
	CleanupInterval   time.Duration `json:"cleanup_interval"`
	
	// 负载均衡配置
	RebalanceEvaluationInterval time.Duration `json:"rebalance_eval_interval"`
	ImbalanceThreshold          float64       `json:"imbalance_threshold"`
	MaxConcurrentMigrations     int           `json:"max_concurrent_migrations"`
	MinMigrationInterval        time.Duration `json:"min_migration_interval"`
	MigrationTimeout            time.Duration `json:"migration_timeout"`
}

// HeartbeatConfig 心跳管理器配置
type HeartbeatConfig struct {
	NodeID            string
	HeartbeatInterval time.Duration
	SuspectTimeout    time.Duration
	DeadTimeout       time.Duration
	CleanupInterval   time.Duration
}

// LoadBalancerConfig 负载均衡管理器配置
type LoadBalancerConfig struct {
	EvaluationInterval      time.Duration // 负载评估时间间隔
	ImbalanceThreshold      float64       // 触发再平衡的负载不平衡阈值(百分比)
	MaxConcurrentMigrations int           // 同时进行的最大迁移任务数
	MinMigrationInterval    time.Duration // 最小迁移间隔时间
	MigrationTimeout        time.Duration // 迁移会话超时时间
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	EnableTLS bool   `json:"enable_tls"` // 是否启用TLS加密
	CertFile  string `json:"cert_file"`  // TLS证书文件路径
	KeyFile   string `json:"key_file"`   // TLS密钥文件路径
	EnableAuth   bool     	   `json:"enable_auth"`    // 是否启用基本认证
	AuthUser     string        `json:"auth_user"`      // 认证用户名
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `json:"level"`  // 日志级别(debug, info, warn, error)
	Output string `json:"output"` // 日志输出位置(stdout, file)
}

// LoadConfig 从文件加载配置
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{}
	// 设置默认值
	config.ShutdownTimeout = 10 * time.Second
	config.Server.ReadTimeout = 30 * time.Second
	config.Server.WriteTimeout = 30 * time.Second

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}

