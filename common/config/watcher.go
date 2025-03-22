package config

import (
	"fmt"
	"log"
	"os"
	"time"
)

// ConfigWatcher 配置监视器
type ConfigWatcher struct {
	configFile string              // 配置文件路径
	lastMod    time.Time           // 最后修改时间
	callback   func(*SystemConfig) // 配置更新回调
	stopChan   chan struct{}       // 停止信号通道
	interval   time.Duration       // 检查间隔
}

const defaultWatchInterval = 30 * time.Second

// 在config包中定义接口
type Reloadable interface {
	ForceReload() error
}

// NewConfigWatcher 创建一个配置文件观察器
// 参数: configFile - 配置文件路径, callback - 配置变更回调
// 返回: 配置观察器实例, 错误信息
func NewConfigWatcher(configFile string, callback func(*SystemConfig)) (*ConfigWatcher, error) {
	info, err := os.Stat(configFile)
	if err != nil {
		return nil, fmt.Errorf("无法获取配置文件信息: %w", err)
	}

	return &ConfigWatcher{
		configFile: configFile,
		callback:   callback,
		stopChan:   make(chan struct{}),
		interval:   defaultWatchInterval,
		lastMod:    info.ModTime(),
	}, nil
}

// Start 开始监视配置文件变更
func (cw *ConfigWatcher) Start() {
	go func() {
		ticker := time.NewTicker(cw.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := cw.checkAndReload(); err != nil {
					// 记录错误但继续运行
					log.Printf("配置重载错误: %v", err)
				}
			case <-cw.stopChan:
				return
			}
		}
	}()
}

// Stop 停止监视配置文件
func (cw *ConfigWatcher) Stop() {
	close(cw.stopChan)
}

// ForceReload 强制重新加载配置
func (cw *ConfigWatcher) ForceReload() error {
	// 修复无限递归调用的问题
	info, err := os.Stat(cw.configFile)
	if err != nil {
		return fmt.Errorf("配置文件状态检查失败: %w", err)
	}

	// 加载新配置
	newConfig, err := LoadSystemConfig(cw.configFile)
	if err != nil {
		return fmt.Errorf("配置重载失败: %w", err)
	}

	// 更新最后修改时间
	cw.lastMod = info.ModTime()

	// 回调通知
	if cw.callback != nil {
		cw.callback(newConfig)
	}

	return nil
}

// checkAndReload 检查配置文件是否变化并重新加载
func (cw *ConfigWatcher) checkAndReload() error {
	info, err := os.Stat(cw.configFile)
	if err != nil {
		return err
	}

	if !info.ModTime().After(cw.lastMod) {
		return nil
	}

	return cw.ForceReload()
}
