package config

import (
	"fmt"
)

// Validate 配置验证函数
func Validate(cfg *ServerConfig) error {
	// 验证 HTTP 端口
	if err := validateHTTPPort(cfg.HTTPPort); err != nil {
		return fmt.Errorf("HTTP端口验证失败: %w", err)
	}

	// 验证心跳间隔
	if err := validateHeartbeatInterval(cfg.HeartbeatInterval); err != nil {
		return fmt.Errorf("心跳间隔验证失败: %w", err)
	}
	// 增加其他配置项验证

	return nil
}

// validateHTTPPort 验证 HTTP 端口是否有效
func validateHTTPPort(port int) error {
	if port < 1024 || port > 65535 {
		return fmt.Errorf("无效的 HTTP 端口: %d，端口号应在1024到65535之间", port)
	}
	return nil
}

// validateHeartbeatInterval 验证心跳间隔是否有效
func validateHeartbeatInterval(interval int) error {
	if interval <= 0 {
		return fmt.Errorf("无效的心跳间隔: %d，心跳间隔应大于0", interval)
	}
	return nil
}

// ...其他验证函数...
