package config

import (
	"time"

	commonconfig "github.com/22827099/DFS_v1/common/config"
)

// Config 元数据服务器配置
type Config struct {
	commonconfig.BaseConfig                // 嵌入基础配置
	Database                DatabaseConfig `json:"database" yaml:"database"`
	Cluster                 ClusterConfig  `json:"cluster" yaml:"cluster"`
	Security                SecurityConfig `json:"security" yaml:"security"`
	ShutdownTimeout         time.Duration  `json:"-" yaml:"-"` // 不从配置文件加载
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type            string `json:"type" yaml:"type"`
	Host            string `json:"host" yaml:"host"`
	Port            int    `json:"port" yaml:"port"`
	User            string `json:"user" yaml:"user"`
	Password        string `json:"password" yaml:"password"`
	Database        string `json:"database" yaml:"database"`
	MaxOpenConns    int    `json:"max_open_conns" yaml:"max_open_conns" default:"20"`
	MaxIdleConns    int    `json:"max_idle_conns" yaml:"max_idle_conns" default:"10"`
	ConnMaxLifetime int    `json:"conn_max_lifetime" yaml:"conn_max_lifetime" default:"3600"`
	DSN             string `json:"dsn" yaml:"dsn"`
}

// ClusterConfig 集群配置
type ClusterConfig struct {
	// 节点配置
	NodeID string `json:"node_id" yaml:"node_id"`
	// 节点地址
	NodeAddress string `json:"node_address" yaml:"node_address"`

	// 集群成员配置
	Peers         []string          `json:"peers" yaml:"peers"`
	PeerAddresses []string          `json:"peer_addresses" yaml:"peer_addresses"`
	PeerMap       map[string]string `json:"-" yaml:"-"`

	// 选举配置
	ElectionTimeout  time.Duration `json:"election_timeout" yaml:"election_timeout" default:"2s"`
	HeartbeatTimeout time.Duration `json:"heartbeat_timeout" yaml:"heartbeat_timeout" default:"500ms"`

	// 心跳配置
	HeartbeatInterval time.Duration `json:"heartbeat_interval" yaml:"heartbeat_interval" default:"1s"`
	SuspectTimeout    time.Duration `json:"suspect_timeout" yaml:"suspect_timeout" default:"3s"`
	DeadTimeout       time.Duration `json:"dead_timeout" yaml:"dead_timeout" default:"10s"`
	CleanupInterval   time.Duration `json:"cleanup_interval" yaml:"cleanup_interval" default:"30s"`

	// 负载均衡配置
	RebalanceEvaluationInterval time.Duration `json:"rebalance_eval_interval" yaml:"rebalance_eval_interval" default:"5m"`
	ImbalanceThreshold          float64       `json:"imbalance_threshold" yaml:"imbalance_threshold" default:"20.0"`
	MaxConcurrentMigrations     int           `json:"max_concurrent_migrations" yaml:"max_concurrent_migrations" default:"5"`
	MinMigrationInterval        time.Duration `json:"min_migration_interval" yaml:"min_migration_interval" default:"30m"`
	MigrationTimeout            time.Duration `json:"migration_timeout" yaml:"migration_timeout" default:"2h"`
}

// HeartbeatConfig 心跳管理器配置
type HeartbeatConfig struct {
	NodeID 		  	  string        `json:"node_id" yaml:"node_id"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval" yaml:"heartbeat_interval" default:"1s"`
	SuspectTimeout    time.Duration `json:"suspect_timeout" yaml:"suspect_timeout" default:"3s"`
	DeadTimeout       time.Duration `json:"dead_timeout" yaml:"dead_timeout" default:"10s"`
	CleanupInterval   time.Duration `json:"cleanup_interval" yaml:"cleanup_interval" default:"30s"`
}

// LoadBalancerConfig 负载均衡管理器配置
type LoadBalancerConfig struct {
	EvaluationInterval      time.Duration `json:"evaluation_interval" yaml:"evaluation_interval" default:"5m"`
	ImbalanceThreshold      float64       `json:"imbalance_threshold" yaml:"imbalance_threshold" default:"20.0"`
	MaxConcurrentMigrations int           `json:"max_concurrent_migrations" yaml:"max_concurrent_migrations" default:"5"`
	MinMigrationInterval    time.Duration `json:"min_migration_interval" yaml:"min_migration_interval" default:"30m"`
	MigrationTimeout        time.Duration `json:"migration_timeout" yaml:"migration_timeout" default:"2h"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	EnableTLS   bool          `json:"enable_tls" yaml:"enable_tls" default:"false"`
	CertFile    string        `json:"cert_file" yaml:"cert_file"`
	KeyFile     string        `json:"key_file" yaml:"key_file"`
	EnableAuth  bool          `json:"enable_auth" yaml:"enable_auth" default:"false"`
	TokenExpiry time.Duration `json:"token_expiry" yaml:"token_expiry" default:"24h"`
	JWTSecret   string        `json:"jwt_secret" yaml:"jwt_secret"`
}
