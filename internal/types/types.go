package types

import "time"

// NodeStatus 表示节点状态
type NodeStatus string
const (
	NodeStatusUnknown NodeStatus = "unknown" // 未知状态
	NodeStatusHealthy NodeStatus = "healthy" // 健康状态
	NodeStatusSuspect NodeStatus = "suspect" // 可疑状态
	NodeStatusDead    NodeStatus = "dead"    // 死亡状态
)

// NodeInfo 表示节点信息
type NodeInfo struct {
	ID       string     	`json:"id"`                 // 节点唯一标识符
	Address  string     	`json:"address"`            // 节点网络地址
	Status   NodeStatus 	`json:"status"`             // 节点当前状态
	IsLeader bool       	`json:"is_leader"`          // 是否为集群leader
	JoinTime int64      	`json:"join_time"`          // 加入集群的时间戳
	LastSeen int64      	`json:"last_seen,omitempty"`// 最后一次检测到的时间戳
	Metrics  *NodeMetrics 	`json:"metrics"`          	// 节点度量指标
}

// NodeMetrics 节点度量指标
type NodeMetrics struct {
	NodeID            string    `json:"node_id"`             // 节点ID
	CPUUsage          float64   `json:"cpu_usage"`           // CPU使用率（百分比）
	MemoryUsage       float64   `json:"memory_usage"`        // 内存使用率（百分比）
	DiskUsage         float64   `json:"disk_usage"`          // 磁盘使用率（百分比）
	TotalStorage      uint64    `json:"total_storage"`       // 存储总容量（字节）
	UsedStorage       uint64    `json:"used_storage"`        // 已用存储（字节）
	FreeStorage       uint64    `json:"free_storage"`        // 可用存储（字节）
	NetworkThroughput float64   `json:"network_throughput"`  // 网络吞吐量（字节/秒）
	IOPS              float64   `json:"iops"`                // IOPS（每秒IO操作数）
	ShardCount        int       `json:"shard_count"`         // 数据分片数量
	HotShardCount     int       `json:"hot_shard_count"`     // 热点数据分片数量
	UpdateTime        time.Time `json:"update_time"`  		 // 更新时间
}