package metadata

import (
	"time"
)

// FileMetadata 表示文件的元数据
type FileMetadata struct {
	FileID     int64     `db:"file_id"`     // 文件ID
	DirID      int64     `db:"dir_id"`      // 所在目录ID
	Name       string    `db:"name"`        // 文件名
	Path       string    `db:"path"`        // 完整路径
	Size       int64     `db:"size"`        // 文件大小(字节)
	Checksum   string    `db:"checksum"`    // 校验和
	Owner      string    `db:"owner"`       // 所有者
	Group      string    `db:"group"`       // 组
	Mode       int32     `db:"mode"`        // 权限模式
	MimeType   string    `db:"mime_type"`   // MIME类型
	Blocks     int32     `db:"blocks"`      // 块数量
	CreateTime time.Time `db:"create_time"` // 创建时间
	ModifyTime time.Time `db:"modify_time"` // 修改时间
	AccessTime time.Time `db:"access_time"` // 访问时间
}

// DirectoryMetadata 表示目录的元数据
type DirectoryMetadata struct {
	DirID      int64     `db:"dir_id"`      // 目录ID
	ParentID   int64     `db:"parent_id"`   // 父目录ID
	Name       string    `db:"name"`        // 目录名称
	Path       string    `db:"path"`        // 完整路径
	Owner      string    `db:"owner"`       // 所有者
	Group      string    `db:"group"`       // 组
	Mode       int32     `db:"mode"`        // 权限模式
	CreateTime time.Time `db:"create_time"` // 创建时间
	ModifyTime time.Time `db:"modify_time"` // 修改时间
	AccessTime time.Time `db:"access_time"` // 访问时间
}


// ChunkMetadata 表示数据块元数据
type ChunkMetadata struct {
	ChunkID    int64  `db:"chunk_id"`
	FileID     int64  `db:"file_id"`
	ChunkIndex int    `db:"chunk_index"`
	Size       int    `db:"size"`
	Checksum   string `db:"checksum"`
}

// ReplicaMetadata 表示副本元数据
type ReplicaMetadata struct {
	ReplicaID int64     `db:"replica_id"`
	ChunkID   int64     `db:"chunk_id"`
	NodeID    string    `db:"node_id"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
}

// PathInfo 表示解析路径后的信息
type PathInfo struct {
	Path       string             // 规范化的路径
	DirID      int64              // 如果是目录，则为目录ID
	FileID     int64              // 如果是文件，则为文件ID
	IsFile     bool               // 是否为文件
	IsDir      bool               // 是否为目录
	Exists     bool               // 路径是否存在
	ParentPath string             // 父目录路径
	Name       string             // 文件或目录名称
	Metadata   interface{}        // 元数据，可能是 DirectoryMetadata 或 FileMetadata
	ParentDir  *DirectoryMetadata // 父目录的元数据
}

// FileSystemEntry 表示文件系统条目
type FileSystemEntry struct {
	Name     string
	Path     string
	IsDir    bool
	Size     int64
	Mode     int
	ModTime  time.Time
	OwnerID  int
	Children []FileSystemEntry // 仅对目录有效
}

// BatchOperation 表示批处理操作
type BatchOperation struct {
	Operation string // create, update, delete
	Path      string
	IsDir     bool
	Metadata  interface{} // FileMetadata 或 DirectoryMetadata
}
