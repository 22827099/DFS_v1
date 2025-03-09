package metadata

import (
	"context"
	"time"
)

// FileType 定义文件类型
type FileType string

const (
	TypeRegular   FileType = "regular"   // 普通文件
	TypeDirectory FileType = "directory" // 目录
	TypeSymlink   FileType = "symlink"   // 符号链接
)

// FileInfo 文件元数据信息
type FileInfo struct {
	Name        string            `json:"name"`              // 文件名称
	Path        string            `json:"path"`              // 文件路径
	Type        FileType          `json:"type"`              // 文件类型
	Size        int64             `json:"size"`              // 文件大小
	MimeType    string            `json:"mime_type,omitempty"` // 文件MIME类型
	ChunkSize   int               `json:"chunk_size"`        // 块大小
	Chunks      []ChunkInfo       `json:"chunks"`            // 块信息列表
	Owner       string            `json:"owner"`             // 所有者
	Permissions string            `json:"permissions"`       // 权限 (例如 "rw-r--r--")
	Replicas    int               `json:"replicas"`          // 副本数
	CreatedAt   time.Time         `json:"created_at"`        // 创建时间
	ModifiedAt  time.Time         `json:"modified_at"`       // 修改时间
	UpdatedAt   time.Time         `json:"updated_at"`        // 更新时间
	Metadata    map[string]string `json:"metadata,omitempty"` // 用户自定义元数据
}

// ChunkInfo 块信息
type ChunkInfo struct {
	Index      int      `json:"index"`               // 块索引
	Size       int64    `json:"size"`                // 块大小
	Offset     int64    `json:"offset"`              // 块在文件中的偏移量
	Checksum   string   `json:"checksum"`            // 块校验和
	Status     string   `json:"status,omitempty"`    // 块状态
	NodeID     string   `json:"node_id,omitempty"`   // 主节点ID
	Locations  []string `json:"locations"`           // 数据节点位置
	Replicas   []string `json:"replicas,omitempty"`  // 副本节点ID列表
}

// DirectoryInfo 目录元数据
type DirectoryInfo struct {
	Name        string            `json:"name"`              // 目录名称
	Path        string            `json:"path"`              // 目录路径
	Owner       string            `json:"owner"`             // 所有者
	Permissions string            `json:"permissions"`       // 权限
	CreatedAt   time.Time         `json:"created_at"`        // 创建时间
	ModifiedAt  time.Time         `json:"modified_at"`       // 修改时间
	UpdatedAt   time.Time         `json:"updated_at"`        // 更新时间
	Metadata    map[string]string `json:"metadata,omitempty"` // 用户自定义元数据
}

// DirectoryEntry 目录项，表示目录下的文件或子目录
type DirectoryEntry struct {
	Name        string    `json:"name"`               // 条目名称
	Path        string    `json:"path"`               // 完整路径
	Type        FileType  `json:"type"`               // 类型 (文件或目录)
	IsDir       bool      `json:"is_dir"`             // 是否是目录
	Size        int64     `json:"size"`               // 文件大小 (如果是文件)
	MimeType    string    `json:"mime_type,omitempty"`   // 仅对文件有效
	ChildCount  int       `json:"child_count,omitempty"` // 仅对目录有效
	CreatedAt   time.Time `json:"created_at"`         // 创建时间
	ModifiedAt  time.Time `json:"modified_at"`        // 修改时间
	UpdatedAt   time.Time `json:"updated_at"`         // 更新时间
}

// Store 定义元数据存储接口
type Store interface {
	// 初始化存储
	Initialize() error
	// 关闭存储
	Close() error
	// 获取文件信息
	GetFileInfo(ctx context.Context, path string) (*FileInfo, error)
	// 创建文件
	CreateFile(ctx context.Context, fileInfo FileInfo) (*FileInfo, error)
	// 更新文件
	UpdateFile(ctx context.Context, path string, updates map[string]interface{}) (*FileInfo, error)
	// 删除文件
	DeleteFile(ctx context.Context, path string) error
	// 列出目录内容
	ListDirectory(ctx context.Context, path string, recursive bool, limit int) ([]DirectoryEntry, error)
	// 创建目录
	CreateDirectory(ctx context.Context, dirInfo DirectoryInfo) (*DirectoryInfo, error)
	// 删除目录
	DeleteDirectory(ctx context.Context, path string, recursive bool) error
}