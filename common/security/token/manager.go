package token

import (
	"sync"
	"time"
)

// TokenManager 提供令牌管理功能
type TokenManager struct {
	maker         Maker
	revokedTokens map[string]time.Time // 存储已撤销的令牌ID及其过期时间
	mu            sync.RWMutex
	cleanupFreq   time.Duration // 清理频率
	stopCleanup   chan struct{}
}

// NewTokenManager 创建一个新的令牌管理器
func NewTokenManager(maker Maker, cleanupFreq time.Duration) *TokenManager {
	tm := &TokenManager{
		maker:         maker,
		revokedTokens: make(map[string]time.Time),
		cleanupFreq:   cleanupFreq,
		stopCleanup:   make(chan struct{}),
	}

	// 启动定期清理过期的已撤销令牌
	go tm.startCleanupTask()

	return tm
}

// RevokeToken 撤销令牌
func (tm *TokenManager) RevokeToken(tokenString string) error {
	payload, err := tm.maker.VerifyToken(tokenString)
	if err != nil {
		return err
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 存储令牌ID和过期时间
	tm.revokedTokens[payload.ID] = payload.ExpiredAt

	return nil
}

// IsRevoked 检查令牌是否已被撤销
func (tm *TokenManager) IsRevoked(tokenID string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	_, revoked := tm.revokedTokens[tokenID]
	return revoked
}

// RefreshToken 刷新令牌
func (tm *TokenManager) RefreshToken(tokenString string, duration time.Duration) (string, error) {
	// 验证当前令牌
	payload, err := tm.maker.VerifyToken(tokenString)
	if err != nil {
		return "", err
	}

	// 检查令牌是否已撤销
	if tm.IsRevoked(payload.ID) {
		return "", ErrInvalidToken
	}

	// 撤销当前令牌
	tm.mu.Lock()
	tm.revokedTokens[payload.ID] = payload.ExpiredAt
	tm.mu.Unlock()

	// 创建新令牌
	return tm.maker.CreateToken(payload.Username, payload.Subject, duration)
}

// 清理过期的已撤销令牌
func (tm *TokenManager) startCleanupTask() {
	ticker := time.NewTicker(tm.cleanupFreq)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tm.cleanup()
		case <-tm.stopCleanup:
			return
		}
	}
}

// 清理过期的已撤销令牌
func (tm *TokenManager) cleanup() {
	now := time.Now()

	tm.mu.Lock()
	defer tm.mu.Unlock()

	for id, expireTime := range tm.revokedTokens {
		if now.After(expireTime) {
			delete(tm.revokedTokens, id)
		}
	}
}

// Stop 停止令牌管理器
func (tm *TokenManager) Stop() {
	close(tm.stopCleanup)
}
