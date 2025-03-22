package config

import (
	"time"

	commonconfig "github.com/22827099/DFS_v1/common/config"
)

// LoadMetaServerConfig 加载元数据服务器配置
func LoadMetaServerConfig(path string) (*Config, error) {
	config := &Config{
		// 设置固定的默认值
		ShutdownTimeout: 10 * time.Second,
	}

	// 使用通用加载器
	if err := commonconfig.LoadConfig(path, config); err != nil {
		return nil, err
	}

	// 加载特定的验证和处理
	if err := validateMetaServerConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// 特定的验证函数
func validateMetaServerConfig(config *Config) error {
	// 元数据服务器特定的验证...
	return nil
}
