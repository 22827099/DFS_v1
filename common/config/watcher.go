package config

import (
	"fmt"
	"time"
)

// ConfigWatcher 配置监视器
type ConfigWatcher struct {
	ConfigPath string
	Config     *SystemConfig
	onChange   func(*SystemConfig)
	stopChan   chan struct{}
}

// NewConfigWatcher 创建新的配置监视器
func NewConfigWatcher(path string, onChange func(*SystemConfig)) (*ConfigWatcher, error) {
	watcher := &ConfigWatcher{
		ConfigPath: path,
		onChange:   onChange,
		stopChan:   make(chan struct{}),
	}

	// 初始加载配置
	config, err := LoadConfigAuto(path)
	if err != nil {
		return nil, err
	}

	watcher.Config = config

	return watcher, nil
}

// Start 开始监视配置文件变化
func (w *ConfigWatcher) Start() {
	go func() {
		ticker := time.NewTicker(30 * time.Second) // 定期检查
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.checkAndReload()
			case <-w.stopChan:
				return
			}
		}
	}()
}

// Stop 停止监视
func (w *ConfigWatcher) Stop() {
	close(w.stopChan)
}

// checkAndReload 检查并重新加载配置
func (w *ConfigWatcher) checkAndReload() {
	if w.ConfigPath == "" {
		return // 避免路径为空
	}

	// 使用 LoadConfigAuto 代替手动分派
	config, err := LoadConfigAuto(w.ConfigPath)
	if err != nil {
		fmt.Printf("重新加载配置失败: %v\n", err)
		return
	}

	// 使用正确的相等比较
	if !configEquals(w.Config, config) {
		w.Config = config
		if w.onChange != nil {
			w.onChange(config)
		}
	}
}
