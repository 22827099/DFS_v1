package auth

import (
	"context"
	"time"

	"DFS_v1/common/security/token"
)

// AuthProvider 是身份验证提供者接口
type AuthProvider interface {
	// Authenticate 验证用户凭证
	Authenticate(ctx context.Context, credentials interface{}) (*UserInfo, error)
}

// TokenConfig 包含令牌相关配置
type TokenConfig struct {
	Secret        string
	Issuer        string
	ExpirationMin int
}

// DefaultAuthenticator 是默认的身份验证器实现
type DefaultAuthenticator struct {
	provider    AuthProvider
	userManager UserManager
	tokenMaker  token.Maker
	tokenConfig TokenConfig
}

// NewDefaultAuthenticator 创建一个新的默认身份验证器
func NewDefaultAuthenticator(
	provider AuthProvider,
	userManager UserManager,
	tokenConfig TokenConfig,
) (*DefaultAuthenticator, error) {
	tokenMaker, err := token.NewJWTMaker(tokenConfig.Secret)
	if err != nil {
		return nil, err
	}

	return &DefaultAuthenticator{
		provider:    provider,
		userManager: userManager,
		tokenMaker:  tokenMaker,
		tokenConfig: tokenConfig,
	}, nil
}

// Authenticate 实现Authenticator接口
func (a *DefaultAuthenticator) Authenticate(ctx context.Context, credentials interface{}) (*UserInfo, error) {
	if a.provider == nil {
		return nil, ErrNoAuthProvider
	}

	return a.provider.Authenticate(ctx, credentials)
}

// VerifyToken 验证令牌
func (a *DefaultAuthenticator) VerifyToken(ctx context.Context, tokenStr string) (*UserInfo, error) {
	// 使用token包验证JWT令牌
	payload, err := a.tokenMaker.VerifyToken(tokenStr)
	if err != nil {
		return nil, err
	}

	// 将payload转换为UserInfo
	userInfo := &UserInfo{
		UserID:   payload.Subject,
		Username: payload.Username,
		// 其他字段从payload中提取
	}

	return userInfo, nil
}

// GenerateToken 生成令牌
func (a *DefaultAuthenticator) GenerateToken(ctx context.Context, user *UserInfo) (string, error) {
	// 设置过期时间
	duration := time.Duration(a.tokenConfig.ExpirationMin) * time.Minute

	// 使用token包生成JWT令牌
	tokenStr, err := a.tokenMaker.CreateToken(user.Username, user.UserID, duration)
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

// RefreshToken 刷新令牌
func (a *DefaultAuthenticator) RefreshToken(ctx context.Context, tokenStr string) (string, error) {
	// 验证现有令牌
	userInfo, err := a.VerifyToken(ctx, tokenStr)
	if err != nil {
		return "", err
	}

	// 生成新令牌
	return a.GenerateToken(ctx, userInfo)
}

// Logout 登出
func (a *DefaultAuthenticator) Logout(ctx context.Context, tokenStr string) error {
	// 可以在这里实现令牌黑名单等功能
	// ...
	return nil
}
