package token

import (
	"time"

	"github.com/google/uuid"
)

// Payload 包含JWT令牌的负载数据
type Payload struct {
	ID        string    `json:"id"`            // 令牌唯一ID
	Username  string    `json:"username"`      // 用户名
	Subject   string    `json:"sub"`           // 主题（通常是用户ID）
	IssuedAt  time.Time `json:"iat"`           // 颁发时间
	ExpiredAt time.Time `json:"exp"`           // 过期时间
	Issuer    string    `json:"iss,omitempty"` // 颁发者
}

// NewPayload 创建一个新的令牌负载
func NewPayload(username, subject string, duration time.Duration, issuer string) (*Payload, error) {
	tokenID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	payload := &Payload{
		ID:        tokenID.String(),
		Username:  username,
		Subject:   subject,
		IssuedAt:  now,
		ExpiredAt: now.Add(duration),
		Issuer:    issuer,
	}

	return payload, nil
}

// Valid 检查负载是否有效（用于jwt-go验证）
func (payload *Payload) Valid() error {
	if time.Now().After(payload.ExpiredAt) {
		return ErrExpiredToken
	}
	return nil
}
