package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Load 配置加载函数
func Load(configPath string) (*ServerConfig, error) {
	v := viper.New()

	// 设置默认值
	v.SetDefault("http_port", 8080)
	v.SetDefault("heartbeat_interval", 15)

	// 配置文件路径设置
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// 如果没有提供路径，使用默认路径
		v.SetConfigName("config")
		v.AddConfigPath("./config")
		v.AddConfigPath(".")
	}

	// 配置环境变量支持
	v.AutomaticEnv()
	v.SetEnvPrefix("DFS_META")
	v.BindEnv("http_port", "HTTP_PORT")

	// 加载配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("加载配置文件失败: %w", err)
	}

	cfg := DefaultConfig()
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return cfg, nil
}
