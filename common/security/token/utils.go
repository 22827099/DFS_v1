package token

import (
	"time"
)

// ExtractTokenMetadata 从令牌中提取元数据
func ExtractTokenMetadata(tokenString string, maker Maker) (*TokenMetadata, error) {
	payload, err := maker.VerifyToken(tokenString)
	if err != nil {
		return nil, err
	}

	metadata := &TokenMetadata{
		UserID:   payload.Subject,
		Username: payload.Username,
		TokenID:  payload.ID,
		ExpireAt: payload.ExpiredAt,
	}

	return metadata, nil
}

// TokenMetadata 包含从令牌提取的元数据
type TokenMetadata struct {
	UserID   string
	Username string
	TokenID  string
	ExpireAt time.Time
}

// IsExpired 检查令牌是否已过期
func (tm *TokenMetadata) IsExpired() bool {
	return time.Now().After(tm.ExpireAt)
}

// RemainingTime 返回令牌剩余有效时间
func (tm *TokenMetadata) RemainingTime() time.Duration {
	if tm.IsExpired() {
		return 0
	}
	return tm.ExpireAt.Sub(time.Now())
}
