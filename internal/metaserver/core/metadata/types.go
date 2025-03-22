package metadata

import (
	"context"
	"time"

	"github.com/22827099/DFS_v1/common/types"
)

// FileInfo 文件元数据信息 - 使用通用基本类型
type FileInfo struct {
	types.BasicFileInfo                // 嵌入基本文件信息
	Type                types.FileType `json:"type"`
	Size                int64          `json:"size"`
	MimeType            string         `json:"mime_type,omitempty"`
	ChunkSize           int            `json:"chunk_size"`
	Chunks              []ChunkInfo    `json:"chunks"`
	Replicas            int            `json:"replicas"`
}

// ChunkInfo 块信息 - 使用通用基本类型
type ChunkInfo struct {
	types.BasicChunkInfo                   // 嵌入基本块信息
	Status               types.ChunkStatus `json:"status,omitempty"`
	NodeID               types.NodeID      `json:"node_id,omitempty"`
	Locations            []string          `json:"locations"`
	Replicas             []types.NodeID    `json:"replicas,omitempty"`
}

// DirectoryInfo 目录元数据 - 使用通用基本类型
type DirectoryInfo struct {
	types.BasicFileInfo // 嵌入基本文件信息
	// 目录特有字段...
}

// DirectoryEntry 目录项 - 使用通用类型
type DirectoryEntry struct {
	Name       string         `json:"name"`
	Path       string         `json:"path"`
	Type       types.FileType `json:"type"`
	IsDir      bool           `json:"is_dir"`
	Size       int64          `json:"size"`
	MimeType   string         `json:"mime_type,omitempty"`
	ChildCount int            `json:"child_count,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	ModifiedAt time.Time      `json:"modified_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// Store 接口保持不变
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
