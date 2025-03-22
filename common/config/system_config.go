package config

import (
	"time"

	"github.com/22827099/DFS_v1/common/types"
)

// SystemConfig 系统配置
type SystemConfig struct {
	NodeID     types.NodeID  `json:"node_id" yaml:"node_id" toml:"node_id" env:"NODE_ID" required:"true"`
	MetaServer string        `json:"meta_server" yaml:"meta_server" toml:"meta_server" env:"META_ADDR" default:"localhost:8080"`
	DataDir    string        `json:"data_dir" yaml:"data_dir" toml:"data_dir" env:"DATA_DIR" default:"./data"`
	ChunkSize  int           `json:"chunk_size" yaml:"chunk_size" toml:"chunk_size" env:"CHUNK_SIZE" default:"1024"`
	Replicas   int           `json:"replicas" yaml:"replicas" toml:"replicas" env:"REPLICAS" default:"2"`
	Logging    LoggingConfig `json:"logging" yaml:"logging" toml:"logging"`
	Server     ServerConfig  `json:"server" yaml:"server" toml:"server"`
}

// ServerConfig 是对 BaseServerConfig 的兼容层
type ServerConfig struct {
	Host         string        `json:"host" yaml:"host" toml:"host" env:"SERVER_HOST" default:"0.0.0.0"`
	Port         int           `json:"port" yaml:"port" toml:"port" env:"SERVER_PORT" default:"8080"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout" toml:"read_timeout" default:"30s"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout" toml:"write_timeout" default:"30s"`
}

// LoadSystemConfig 加载系统配置
func LoadSystemConfig(path string) (*SystemConfig, error) {
	config := &SystemConfig{}
	if err := LoadConfig(path, config); err != nil {
		return nil, err
	}
	return config, nil
}
