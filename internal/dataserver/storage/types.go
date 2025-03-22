package storage

import (
	"time"

	"github.com/22827099/DFS_v1/common/types"
)

// StorageInfo 存储信息
type StorageInfo struct {
	NodeID        types.NodeID `json:"node_id"`
	TotalSpace    uint64       `json:"total_space"`
	UsedSpace     uint64       `json:"used_space"`
	ReservedSpace uint64       `json:"reserved_space"`
	Dirs          []string     `json:"dirs"`
}

// ChunkMetadata 数据节点的块元数据
type ChunkMetadata struct {
	types.BasicChunkInfo                   // 嵌入基本块信息
	FilePath             string            `json:"file_path"`
	Status               types.ChunkStatus `json:"status"`
	StoragePath          string            `json:"storage_path"`
	Verified             time.Time         `json:"verified_at"`
}
