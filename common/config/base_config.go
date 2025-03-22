package config

import (
	"time"

	"github.com/22827099/DFS_v1/common/types"
)

// NodeIdentity 节点标识配置
type NodeIdentity struct {
	NodeID      types.NodeID `json:"id" yaml:"id" toml:"id" env:"NODE_ID" required:"true"`
	Role    string       `json:"role" yaml:"role" toml:"role" env:"NODE_ROLE" default:"member"`
	DataDir string       `json:"data_dir" yaml:"data_dir" toml:"data_dir" env:"DATA_DIR" default:"./data"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level   string `json:"level" yaml:"level" toml:"level" env:"LOG_LEVEL" default:"info"`
	Console bool   `json:"console" yaml:"console" toml:"console" env:"LOG_CONSOLE" default:"true"`
	File    string `json:"file" yaml:"file" toml:"file" env:"LOG_FILE" default:"logs/app.log"`
}

// BaseServerConfig 通用服务器配置
type BaseServerConfig struct {
	Host         string        `json:"host" yaml:"host" toml:"host" env:"SERVER_HOST" default:"0.0.0.0"`
	Port         int           `json:"port" yaml:"port" toml:"port" env:"SERVER_PORT" default:"8080"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout" toml:"read_timeout" default:"30s"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout" toml:"write_timeout" default:"30s"`
	EnableCORS   bool          `json:"enable_cors" yaml:"enable_cors" toml:"enable_cors" default:"false"`
	AllowOrigins []string      `json:"allow_origins" yaml:"allow_origins" toml:"allow_origins"`
}

// ConsensusConfig 共识算法基础配置
type ConsensusConfig struct {
	Protocol           string        `json:"protocol" yaml:"protocol" toml:"protocol" default:"raft"`
	DataDir            string        `json:"data_dir" yaml:"data_dir" toml:"data_dir"`
	SnapshotThreshold  int           `json:"snapshot_threshold" yaml:"snapshot_threshold" default:"10000"`
	CompactionInterval time.Duration `json:"compaction_interval" yaml:"compaction_interval" default:"24h"`
}

// BaseConfig 所有服务基础配置
type BaseConfig struct {
	Node      NodeIdentity     `json:"node" yaml:"node" toml:"node"`
	Logging   LoggingConfig    `json:"logging" yaml:"logging" toml:"logging"`
	Server    BaseServerConfig `json:"server" yaml:"server" toml:"server"`
	Consensus ConsensusConfig  `json:"consensus" yaml:"consensus" toml:"consensus"`
}
