package config

import (
    // 导入其他必要的包
    "github.com/22827099/DFS_v1/internal/metaserver/config" // 导入定义 ClusterConfig 的包
)

// LoggingConfig 日志配置结构体
type LoggingConfig struct {
    Level   string `yaml:"level" json:"level" toml:"level" env:"LOG_LEVEL" default:"info"`
    Console bool   `yaml:"console" json:"console" toml:"console" env:"LOG_CONSOLE" default:"true"`
    File    string `yaml:"file" json:"file" toml:"file" env:"LOG_FILE" default:"logs/app.log"`
}

// SystemConfig 配置结构体
type SystemConfig struct {
    NodeID     string        `yaml:"node_id" json:"node_id" toml:"node_id" env:"NODE_ID" required:"true"`
    MetaServer string        `yaml:"meta_server" json:"meta_server" toml:"meta_server" env:"META_ADDR" default:"localhost:8080"`
    DataDir    string        `yaml:"data_dir" json:"data_dir" toml:"data_dir" env:"DATA_DIR" default:"./data"`
    ChunkSize  int           `yaml:"chunk_size" json:"chunk_size" toml:"chunk_size" env:"CHUNK_SIZE" default:"1024"`
    Replicas   int           `yaml:"replicas" json:"replicas" toml:"replicas" env:"REPLICAS" default:"2"`
    Logging    LoggingConfig `yaml:"logging" json:"logging" toml:"logging"`
    Cluster config.ClusterConfig `json:"cluster" json:"cluster" toml:"cluster"`
}