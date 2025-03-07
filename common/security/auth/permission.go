package auth

import (
	"context"
	"strings"
)

// Permission 表示系统中的权限
type Permission struct {
	Resource string // 资源路径，如"file:/data/users"
	Action   string // 操作，如"read", "write", "delete"
}

// PermissionRule 表示权限规则
type PermissionRule struct {
	Role    Role
	Pattern string // 资源模式，支持通配符，如"file:/data/*"
	Actions []string
	Allow   bool // true表示允许，false表示拒绝
}

// SimplePermissionChecker 是基于规则的权限检查器
type SimplePermissionChecker struct {
	rules []PermissionRule
}

// NewSimplePermissionChecker 创建一个新的权限检查器
func NewSimplePermissionChecker(rules []PermissionRule) *SimplePermissionChecker {
	return &SimplePermissionChecker{rules: rules}
}

// HasPermission 检查用户是否有指定权限
func (c *SimplePermissionChecker) HasPermission(ctx context.Context, user *UserInfo, resource string, action string) bool {
	if user == nil {
		return false
	}

	// 系统角色拥有所有权限
	for _, role := range user.Roles {
		if role == RoleAdmin || role == RoleSystem {
			return true
		}
	}

	// 检查用户角色匹配的规则
	for _, rule := range c.rules {
		// 检查角色是否匹配
		roleMatch := false
		for _, userRole := range user.Roles {
			if userRole == rule.Role {
				roleMatch = true
				break
			}
		}

		if !roleMatch {
			continue
		}

		// 检查资源是否匹配
		if matchResourcePattern(resource, rule.Pattern) {
			// 检查操作是否匹配
			for _, allowedAction := range rule.Actions {
				if allowedAction == "*" || allowedAction == action {
					return rule.Allow
				}
			}
		}
	}

	// 默认拒绝
	return false
}

// GetUserPermissions 获取用户所有权限
func (c *SimplePermissionChecker) GetUserPermissions(ctx context.Context, user *UserInfo) ([]Permission, error) {
	// 实现获取用户所有权限的逻辑
	// ...
	return nil, nil
}

// matchResourcePattern 检查资源是否匹配模式
func matchResourcePattern(resource, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// 简单的通配符匹配
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(resource, prefix)
	}

	return resource == pattern
}
