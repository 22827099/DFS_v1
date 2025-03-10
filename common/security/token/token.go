package token

import (
	"errors"
	"time"
	"strings"
	"net/http"
)

// 常见错误定义
var (
	ErrInvalidToken         = errors.New("令牌无效")
	ErrExpiredToken         = errors.New("令牌已过期")
	ErrTokenCreateFailed    = errors.New("创建令牌失败")
	ErrInvalidSigningMethod = errors.New("无效的签名方法")
)

// Maker 是令牌创建和验证的接口
type Maker interface {
	// CreateToken 创建一个新令牌，包含用户名、主题（通常是用户ID）和有效期
	CreateToken(username, subject string, duration time.Duration) (string, error)

	// VerifyToken 验证令牌的有效性，并返回包含在令牌中的负载
	VerifyToken(token string) (*Payload, error)
}

// extractTokenFromRequest 从请求中提取认证令牌
func ExtractTokenFromRequest(r *http.Request) string {
    // 首先尝试从Authorization头部获取
    authHeader := r.Header.Get("Authorization")
    if authHeader != "" {
        // Bearer token格式
        if strings.HasPrefix(authHeader, "Bearer ") {
            return strings.TrimPrefix(authHeader, "Bearer ")
        }
        return authHeader
    }
    
    // 如果头部没有，尝试从cookie获取
    cookie, err := r.Cookie("auth_token")
    if err == nil && cookie != nil {
        return cookie.Value
    }
    
    // 最后尝试从URL参数获取
    return r.URL.Query().Get("token")
}