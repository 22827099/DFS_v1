package lock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/22827099/DFS_v1/common/logging"
)

// LockType 表示锁类型
type LockType int

const (
	ReadLock LockType = iota
	WriteLock
	IntentRead
	IntentWrite
)

// LockInfo 表示锁信息
type LockInfo struct {
	Owner     string    // 锁拥有者标识
	Type      LockType  // 锁类型
	Timestamp time.Time // 获取时间
}

// Manager 锁管理器
type Manager struct {
	logger      logging.Logger
	pathLocks   sync.Map // 路径到锁的映射
	waitTimeout time.Duration
	lockTimeout time.Duration
	cleanupCh   chan struct{}
}

// NewManager 创建锁管理器
func NewManager(logger logging.Logger) (*Manager, error) {
	return &Manager{
		logger:      logger,
		waitTimeout: 30 * time.Second,    // 等待锁的超时时间
		lockTimeout: 5 * time.Minute,     // 锁的最长持有时间
		cleanupCh:   make(chan struct{}), // 清理通道
	}, nil
}

// Start 启动锁管理器
func (m *Manager) Start() error {
	m.logger.Info("启动锁管理器")
	go m.cleanupExpiredLocks()
	return nil
}

// Stop 停止锁管理器
func (m *Manager) Stop(ctx context.Context) error {
	m.logger.Info("停止锁管理器")
	close(m.cleanupCh)
	return nil
}

// 清理过期的锁
func (m *Manager) cleanupExpiredLocks() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			m.pathLocks.Range(func(key, value interface{}) bool {
				path := key.(string)
				lockInfo := value.(*LockInfo)

				if now.Sub(lockInfo.Timestamp) > m.lockTimeout {
					m.logger.Warn("发现过期锁: 路径=%s, 拥有者=%s, 类型=%v, 持有时间=%v",
						path, lockInfo.Owner, lockInfo.Type, now.Sub(lockInfo.Timestamp))
					m.pathLocks.Delete(path)
				}
				return true
			})
		case <-m.cleanupCh:
			return
		}
	}
}

// Lock 获取锁
func (m *Manager) Lock(ctx context.Context, path string, lockType LockType, owner string) error {
	deadline := time.Now().Add(m.waitTimeout)

	for {
		// 尝试获取锁
		if m.tryLock(path, lockType, owner) {
			return nil
		}

		// 检查是否超时
		if time.Now().After(deadline) {
			return fmt.Errorf("获取路径锁超时: %s", path)
		}

		// 等待一段时间后重试
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// 继续尝试
		}
	}
}

// Unlock 释放锁
func (m *Manager) Unlock(path string, owner string) {
	value, ok := m.pathLocks.Load(path)
	if !ok {
		m.logger.Warn("尝试释放不存在的锁: %s", path)
		return
	}

	lockInfo := value.(*LockInfo)
	if lockInfo.Owner != owner {
		m.logger.Warn("尝试释放他人的锁: 路径=%s, 请求者=%s, 拥有者=%s",
			path, owner, lockInfo.Owner)
		return
	}

	m.pathLocks.Delete(path)
}

// 尝试获取锁
func (m *Manager) tryLock(path string, lockType LockType, owner string) bool {
	currentLock, loaded := m.pathLocks.LoadOrStore(
		path,
		&LockInfo{
			Owner:     owner,
			Type:      lockType,
			Timestamp: time.Now(),
		},
	)

	// 如果没有已存在的锁，直接成功
	if !loaded {
		return true
	}

	// 检查是否可以共享锁
	existingLock := currentLock.(*LockInfo)
	if canShareLock(existingLock.Type, lockType) && existingLock.Owner == owner {
		// 允许同一拥有者升级或共享锁
		return true
	}

	return false
}

// 检查两个锁是否可以共享
func canShareLock(existing, requested LockType) bool {
	// 如果两者都是读锁，可以共享
	if existing == ReadLock && requested == ReadLock {
		return true
	}
	return false
}

// IsLocked 检查路径是否被锁定
func (m *Manager) IsLocked(path string) bool {
	_, locked := m.pathLocks.Load(path)
	return locked
}
