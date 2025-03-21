package config

import (
	"time"

	"github.com/22827099/DFS_v1/common/types"
	"github.com/22827099/DFS_v1/internal/metaserver/config"
)

// LoggingConfig 日志配置结构体
type LoggingConfig struct {
	Level   string `yaml:"level" json:"level" toml:"level" env:"LOG_LEVEL" default:"info"`
	Console bool   `yaml:"console" json:"console" toml:"console" env:"LOG_CONSOLE" default:"true"`
	File    string `yaml:"file" json:"file" toml:"file" env:"LOG_FILE" default:"logs/app.log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
}

// SystemConfig 配置结构体
type SystemConfig struct {
	NodeID     types.NodeID         `yaml:"node_id" json:"node_id" toml:"node_id" env:"NODE_ID" required:"true"`
	MetaServer string               `yaml:"meta_server" json:"meta_server" toml:"meta_server" env:"META_ADDR" default:"localhost:8080"`
	DataDir    string               `yaml:"data_dir" json:"data_dir" toml:"data_dir" env:"DATA_DIR" default:"./data"`
	ChunkSize  int                  `yaml:"chunk_size" json:"chunk_size" toml:"chunk_size" env:"CHUNK_SIZE" default:"1024"`
	Replicas   int                  `yaml:"replicas" json:"replicas" toml:"replicas" env:"REPLICAS" default:"2"`
	Logging    LoggingConfig        `yaml:"logging" json:"logging" toml:"logging"`
	Cluster    config.ClusterConfig `yaml:"cluster" json:"cluster" toml:"cluster"`
	Version    string               `json:"version"`
	Server     ServerConfig         `yaml:"server" json:"server"`
	Consensus  ConsensusConfig      `yaml:"consensus" json:"consensus"`
}

// ConsensusConfig 共识算法配置
type ConsensusConfig struct {
	Protocol           string        `yaml:"protocol" json:"protocol"`
	DataDir            string        `yaml:"data_dir" json:"data_dir"`
	SnapshotThreshold  int           `yaml:"snapshot_threshold" json:"snapshot_threshold"`
	CompactionInterval time.Duration `yaml:"compaction_interval" json:"compaction_interval"`
}
