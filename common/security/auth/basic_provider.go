package auth

import (
	"context"
	"errors"
)

// BasicCredentials 包含基本认证凭证
type BasicCredentials struct {
	Username string
	Password string
}

// BasicAuthProvider 实现基于用户名和密码的认证
type BasicAuthProvider struct {
	userManager UserManager
}

// NewBasicAuthProvider 创建新的基本认证提供者
func NewBasicAuthProvider(userManager UserManager) *BasicAuthProvider {
	return &BasicAuthProvider{
		userManager: userManager,
	}
}

// Authenticate 验证用户凭证
func (p *BasicAuthProvider) Authenticate(ctx context.Context, credentials interface{}) (*UserInfo, error) {
	// 转换凭证类型
	creds, ok := credentials.(*BasicCredentials)
	if !ok {
		return nil, errors.New("无效的凭证类型")
	}

	// 验证用户名和密码
	valid, user, err := p.userManager.VerifyPassword(ctx, creds.Username, creds.Password)
	if err != nil {
		return nil, err
	}

	if !valid || !user.Active {
		return nil, ErrInvalidCredentials
	}

	// 创建用户信息
	userInfo := &UserInfo{
		UserID:    user.ID,
		Username:  user.Username,
		Roles:     user.Roles,
		ExtraData: user.ExtraData,
	}

	return userInfo, nil
}
