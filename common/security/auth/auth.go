package auth

import (
	"context"
	"errors"
	"net/http"
)

// 常见错误定义
var (
	ErrInvalidCredentials = errors.New("无效的凭证")
	ErrPermissionDenied   = errors.New("权限被拒绝")
	ErrExpiredToken       = errors.New("令牌已过期")
	ErrNoAuthProvider     = errors.New("未配置认证提供者")
)

// Role 表示用户在系统中的角色
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleUser   Role = "user"
	RoleGuest  Role = "guest"
	RoleSystem Role = "system"
)

// UserInfo 包含用户的认证信息
type UserInfo struct {
	UserID    string
	Username  string
	Roles     []Role
	ExtraData map[string]interface{}
}

// Authenticator 是身份认证的主要接口
type Authenticator interface {
	// Authenticate 验证用户凭证，成功返回UserInfo，失败返回错误
	Authenticate(ctx context.Context, credentials interface{}) (*UserInfo, error)

	// VerifyToken 验证令牌的有效性，成功返回UserInfo，失败返回错误
	VerifyToken(ctx context.Context, token string) (*UserInfo, error)

	// GenerateToken 为用户生成令牌
	GenerateToken(ctx context.Context, user *UserInfo) (string, error)

	// RefreshToken 刷新现有令牌
	RefreshToken(ctx context.Context, token string) (string, error)

	// Logout 使令牌失效
	Logout(ctx context.Context, token string) error
}

// PermissionChecker 负责检查用户的权限
type PermissionChecker interface {
	// HasPermission 检查用户是否有特定权限
	HasPermission(ctx context.Context, user *UserInfo, resource string, action string) bool

	// GetUserPermissions 获取用户所有权限
	GetUserPermissions(ctx context.Context, user *UserInfo) ([]Permission, error)
}

// AuthMiddleware 创建HTTP认证中间件
func AuthMiddleware(auth Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 从请求中获取令牌
			token := extractTokenFromRequest(r)
			if token == "" {
				http.Error(w, "未提供认证令牌", http.StatusUnauthorized)
				return
			}

			// 验证令牌
			userInfo, err := auth.VerifyToken(r.Context(), token)
			if err != nil {
				if errors.Is(err, ErrExpiredToken) {
					http.Error(w, "令牌已过期", http.StatusUnauthorized)
				} else {
					http.Error(w, "无效的认证令牌", http.StatusUnauthorized)
				}
				return
			}

			// 将用户信息添加到请求上下文
			ctx := context.WithValue(r.Context(), userContextKey, userInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// 从请求中提取认证令牌
func extractTokenFromRequest(r *http.Request) string {
	// 首先尝试从Authorization头获取
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}

	// 然后尝试从cookie获取
	cookie, err := r.Cookie("auth_token")
	if err == nil {
		return cookie.Value
	}

	// 最后尝试从URL参数获取
	return r.URL.Query().Get("token")
}

// 用于在上下文中存储用户信息的键
type contextKey int

const userContextKey contextKey = 0

// GetUserFromContext 从上下文中获取用户信息
func GetUserFromContext(ctx context.Context) (*UserInfo, bool) {
	user, ok := ctx.Value(userContextKey).(*UserInfo)
	return user, ok
}

// WithUserContext 将用户信息添加到请求上下文中
func WithUserContext(ctx context.Context, user *UserInfo) context.Context {
    return context.WithValue(ctx, userContextKey, user)
}
