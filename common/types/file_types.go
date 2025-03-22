package types

import "time"

// FileType 定义文件类型
type FileType string

const (
	TypeRegular   FileType = "regular"   // 普通文件
	TypeDirectory FileType = "directory" // 目录
	TypeSymlink   FileType = "symlink"   // 符号链接
)

// FilePermission 定义文件权限
type FilePermission string

// BasicFileInfo 基本文件信息(所有文件系统实体的共同特性)
type BasicFileInfo struct {
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	Owner       string            `json:"owner"`
	Permissions string            `json:"permissions"`
	CreatedAt   time.Time         `json:"created_at"`
	ModifiedAt  time.Time         `json:"modified_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
