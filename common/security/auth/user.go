package auth

import (
	"context"
	"errors"
	"sync"
	"time"
)

// User 表示系统中的用户
type User struct {
	ID           string
	Username     string
	PasswordHash string
	Roles        []Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLogin    time.Time
	Active       bool
	ExtraData    map[string]interface{}
}

// UserManager 提供用户管理功能
type UserManager interface {
	// GetUser 通过ID获取用户
	GetUser(ctx context.Context, userID string) (*User, error)

	// GetUserByUsername 通过用户名获取用户
	GetUserByUsername(ctx context.Context, username string) (*User, error)

	// CreateUser 创建新用户
	CreateUser(ctx context.Context, user *User) error

	// UpdateUser 更新用户信息
	UpdateUser(ctx context.Context, user *User) error

	// DeleteUser 删除用户
	DeleteUser(ctx context.Context, userID string) error

	// VerifyPassword 验证用户密码
	VerifyPassword(ctx context.Context, username, password string) (bool, *User, error)

	// UpdatePassword 更新用户密码
	UpdatePassword(ctx context.Context, userID, newPassword string) error
}

// InMemoryUserManager 是一个内存实现的UserManager
type InMemoryUserManager struct {
	users  map[string]*User
	byName map[string]string // username -> userID
	mu     sync.RWMutex
}

// NewInMemoryUserManager 创建一个新的内存用户管理器
func NewInMemoryUserManager() *InMemoryUserManager {
	return &InMemoryUserManager{
		users:  make(map[string]*User),
		byName: make(map[string]string),
	}
}

// 实现UserManager接口
func (m *InMemoryUserManager) GetUser(ctx context.Context, userID string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if user, ok := m.users[userID]; ok {
		return user, nil
	}
	return nil, errors.New("用户不存在")
}

func (m *InMemoryUserManager) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userID, ok := m.byName[username]
	if !ok {
		return nil, errors.New("用户不存在")
	}
	return m.users[userID], nil
}

func (m *InMemoryUserManager) CreateUser(ctx context.Context, user *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查用户名是否已存在
	if _, ok := m.byName[user.Username]; ok {
		return errors.New("用户名已存在")
	}

	// 设置创建时间
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	user.UpdatedAt = time.Now()

	m.users[user.ID] = user
	m.byName[user.Username] = user.ID

	return nil
}

// 其他方法实现...
