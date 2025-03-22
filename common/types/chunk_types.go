package types

// ChunkStatus 表示数据块的状态
type ChunkStatus string

const (
	ChunkStatusNormal    ChunkStatus = "normal"    // 正常
	ChunkStatusCorrupted ChunkStatus = "corrupted" // 损坏
	ChunkStatusMissing   ChunkStatus = "missing"   // 缺失
	ChunkStatusRepairing ChunkStatus = "repairing" // 修复中
)

// BasicChunkInfo 基本块信息
type BasicChunkInfo struct {
	Index    int    `json:"index"`
	Size     int64  `json:"size"`
	Offset   int64  `json:"offset"`
	Checksum string `json:"checksum"`
}
