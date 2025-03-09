package cluster

import (
	"context"

	"github.com/22827099/DFS_v1/internal/types"

)

// Manager 定义集群管理的基本接口
type Manager interface {
	Start() error                                               // 启动集群管理服务
	Stop(ctx context.Context) error                             // 停止集群管理服务
	IsLeader() bool                                             // 检查当前节点是否为leader
	GetCurrentLeader() string                                   // 获取当前leader的节点ID
	RegisterNode(nodeID string)                                 // 注册新节点到集群
	UnregisterNode(nodeID string)                               // 从集群中注销节点
	AddPeer(peerID string) error                                // 添加一个新的peer节点
	RemovePeer(peerID string) error                             // 移除一个peer节点
	TriggerRebalance()                                          // 触发集群重平衡
	GetRebalanceStatus() map[string]interface{}                 // 获取重平衡状态信息
	UpdateNodeMetrics(nodeID string, metrics *types.NodeMetrics) // 更新节点指标信息
	LeaderChangeChan() <-chan string                            // 返回leader变更通知通道
	ListNodes(ctx context.Context) ([]types.NodeInfo, error) // 列出所有集群节点
	GetNodeInfo(ctx context.Context, nodeID string) (*types.NodeInfo, error)
    GetLeader(ctx context.Context) (*types.NodeInfo, error) // 注意同时添加handleGetLeader所需方法
}
