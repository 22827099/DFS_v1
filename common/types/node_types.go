package types

import "fmt"

// NodeID 是分布式系统中节点的唯一标识
// 在整个系统中统一使用此类型表示节点标识
type NodeID string

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
	NodeID   NodeID       `json:"id"`                  // 节点唯一标识符
	Address  string       `json:"address"`             // 节点网络地址
	Status   NodeStatus   `json:"status"`              // 节点当前状态
	IsLeader bool         `json:"is_leader"`           // 是否为集群leader
	JoinTime int64        `json:"join_time"`           // 加入集群的时间戳
	LastSeen int64        `json:"last_seen,omitempty"` // 最后一次检测到的时间戳
	Metrics  *NodeMetrics `json:"metrics"`             // 节点度量指标
}

// String 返回字符串表示
func (n NodeID) String() string {
	return string(n)
}

// IsEmpty 检查NodeID是否为空
func (n NodeID) IsEmpty() bool {
	return n == ""
}

// Equals 比较两个NodeID是否相等
func (n NodeID) Equals(other NodeID) bool {
	return n == other
}

// ToLogString 返回用于日志的字符串表示
func (n NodeID) ToLogString() string {
	if n.IsEmpty() {
		return "[空节点]"
	}
	return fmt.Sprintf("[节点:%s]", string(n))
}
