package middleware

import (
	"net/http"
	"strings"

	"github.com/22827099/DFS_v1/common/errors"
	nethttp "github.com/22827099/DFS_v1/common/network/http"
	"github.com/22827099/DFS_v1/common/security/auth"
	"github.com/22827099/DFS_v1/internal/metaserver/server/api"
	"github.com/22827099/DFS_v1/common/security/token"
)

// AuthService 认证服务接口
type AuthService interface {
	// VerifyToken 验证用户令牌
	VerifyToken(token string) (*auth.UserInfo, error)
	// HasPermission 检查权限
	HasPermission(user *auth.UserInfo, resource string, action string) bool
}

// Auth 创建认证中间件
func Auth(authService AuthService) nethttp.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 跳过公开路径检查
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// 从请求中提取令牌
			token := token.ExtractTokenFromRequest(r)
			if token == "" {
				api.RespondError(w, r, http.StatusUnauthorized,
					errors.New(errors.Unauthenticated, "未提供认证令牌"))
				return
			}

			// 验证令牌
			user, err := authService.VerifyToken(token)
			if err != nil {
				api.RespondError(w, r, http.StatusUnauthorized,
					errors.New(errors.Unauthenticated, "无效的认证令牌: %v", err))
				return
			}

			// 检查资源访问权限
			// 这里可以根据路径和HTTP方法确定需要的权限
			resource := r.URL.Path
			action := auth.GetActionFromMethod(r.Method)
			if !authService.HasPermission(user, resource, action) {
				api.RespondError(w, r, http.StatusForbidden,
					errors.New(errors.PermissionDenied, "无权访问此资源"))
				return
			}

			// 将用户信息添加到请求上下文
			ctx := auth.WithUserContext(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// 其他辅助函数...
// isPublicPath 检查路径是否是公开的（不需要认证）
func isPublicPath(path string) bool {
    publicPaths := []string{
        "/health",
        "/metrics",
        "/api/v1/auth/login",
        "/api/v1/auth/register",
    }
    
    for _, publicPath := range publicPaths {
        if strings.HasPrefix(path, publicPath) {
            return true
        }
    }
    return false
}