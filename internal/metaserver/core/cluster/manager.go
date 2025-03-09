package cluster

import (
	"context"
	"sync"
	"time"

	"github.com/22827099/DFS_v1/common/logging"
	"github.com/22827099/DFS_v1/internal/metaserver/core/cluster/election"
	"github.com/22827099/DFS_v1/internal/metaserver/core/cluster/heartbeat"
	"github.com/22827099/DFS_v1/internal/metaserver/core/cluster/rebalance"
)

// NodeState 表示集群节点的当前状态
type NodeState string

const (
	NodeStateUnknown   NodeState = "unknown"
	NodeStateHealthy   NodeState = "healthy"
	NodeStateSuspect   NodeState = "suspect"
	NodeStateDead      NodeState = "dead"
	NodeStateDeparting NodeState = "departing"
	NodeStateJoining   NodeState = "joining"
)

// NodeInfo 存储节点信息
type NodeInfo struct {
	ID          string
	Address     string
	State       NodeState
	IsLeader    bool
	LastContact time.Time
	Tags        map[string]string
	Stats       map[string]interface{}
}

// ClusterManager 管理集群相关的所有功能
type ClusterManager struct {
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	nodeID string
	nodes  map[string]*NodeInfo

	// 子模块
	election  *election.Manager
	heartbeat *heartbeat.Manager
	rebalance *rebalance.Manager

	// 通知通道
	stateChangeCh  chan *NodeInfo
	leaderChangeCh chan string

	logger logging.Logger
}

// Config 集群管理器配置
type Config struct {
	NodeID          string
	NodeAddress     string
	ElectionConfig  *election.Config
	HeartbeatConfig *heartbeat.Config
	RebalanceConfig *rebalance.Config
}

// NewManager 创建一个新的集群管理器
func NewManager(cfg *Config, logger logging.Logger) (*ClusterManager, error) {
	if logger == nil {
		// 使用正确的创建Logger的方式
		logger = logging.NewLogger() 
		
		// 如果需要设置前缀或上下文
		logger = logger.WithContext(map[string]interface{}{
			"component": "cluster",
			"nodeID": cfg.NodeID,
		})
	}

	ctx, cancel := context.WithCancel(context.Background())

	cm := &ClusterManager{
		nodeID:         cfg.NodeID,
		nodes:          make(map[string]*NodeInfo),
		stateChangeCh:  make(chan *NodeInfo, 100),
		leaderChangeCh: make(chan string, 10),
		ctx:            ctx,
		cancel:         cancel,
		logger:         logger,
	}

	// 创建心跳管理器
	var err error
	cm.heartbeat, err = heartbeat.NewManager(cfg.HeartbeatConfig, logger)
	if err != nil {
		cancel()
		return nil, err
	}

	// 创建选举管理器
	cm.election, err = election.NewManager(cfg.ElectionConfig, logger)
	if err != nil {
		cancel()
		return nil, err
	}

	// 创建负载均衡管理器
	cm.rebalance, err = rebalance.NewManager(cfg.RebalanceConfig, logger)
	if err != nil {
		cancel()
		return nil, err
	}

	return cm, nil
}

// Start 启动集群管理器及其所有子模块
func (cm *ClusterManager) Start() error {
	cm.logger.Info("启动集群管理器")

	// 启动心跳检测
	if err := cm.heartbeat.Start(); err != nil {
		return err
	}

	// 启动领导选举
	if err := cm.election.Start(); err != nil {
		cm.heartbeat.Stop()
		return err
	}

	// 启动负载均衡
	if err := cm.rebalance.Start(); err != nil {
		cm.heartbeat.Stop()
		cm.election.Stop()
		return err
	}

	// 启动状态处理协程
	go cm.handleStateChanges()

	return nil
}

// Stop 停止集群管理器及其所有子模块
func (cm *ClusterManager) Stop() error {
	cm.logger.Info("停止集群管理器")
	cm.cancel()

	// 按顺序停止各个子系统
	cm.rebalance.Stop()
	cm.election.Stop()
	cm.heartbeat.Stop()

	return nil
}

// GetNodes 获取当前所有节点信息
func (cm *ClusterManager) GetNodes() []*NodeInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	nodes := make([]*NodeInfo, 0, len(cm.nodes))
	for _, node := range cm.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetLeaderNode 获取当前的领导节点
func (cm *ClusterManager) GetLeaderNode() *NodeInfo {
	leaderID := cm.election.GetCurrentLeader()
	if leaderID == "" {
		return nil
	}

	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.nodes[leaderID]
}

// IsLeader 检查当前节点是否是领导节点
func (cm *ClusterManager) IsLeader() bool {
	return cm.election.IsLeader()
}

// handleStateChanges 处理节点状态变更
func (cm *ClusterManager) handleStateChanges() {
	heartbeatCh := cm.heartbeat.StateChangeChan()
	electionCh := cm.election.LeaderChangeChan()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case state := <-heartbeatCh:
			cm.updateNodeState(state.NodeID, convertNodeState(state.State))
		case leaderID := <-electionCh:
			cm.updateLeader(leaderID)
		}
	}
}

// updateNodeState 更新节点状态
func (cm *ClusterManager) updateNodeState(nodeID string, state NodeState) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if node, exists := cm.nodes[nodeID]; exists {
		node.State = state
		node.LastContact = time.Now()
	} else {
		cm.nodes[nodeID] = &NodeInfo{
			ID:          nodeID,
			State:       state,
			LastContact: time.Now(),
			Tags:        make(map[string]string),
			Stats:       make(map[string]interface{}),
		}
	}

	cm.stateChangeCh <- cm.nodes[nodeID]

	// 如果节点死亡且是领导者，触发新一轮选举
	if state == NodeStateDead && cm.nodes[nodeID].IsLeader {
		cm.election.TriggerElection()
	}

	// 触发负载均衡检查
	if state == NodeStateJoining || state == NodeStateDeparting || state == NodeStateDead {
		cm.rebalance.TriggerRebalance()
	}
}

// updateLeader 更新领导节点
func (cm *ClusterManager) updateLeader(leaderID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for id, node := range cm.nodes {
		node.IsLeader = (id == leaderID)
	}

	cm.leaderChangeCh <- leaderID
	cm.logger.Info("领导节点已更新", "nodeID", leaderID)
}

// convertNodeState 将心跳包中的节点状态转换为集群管理器使用的节点状态
func convertNodeState(state heartbeat.NodeState) NodeState {
    switch state {
    case heartbeat.NodeStateHealthy:
        return NodeStateHealthy
    case heartbeat.NodeStateSuspect:
        return NodeStateSuspect
    case heartbeat.NodeStateDead:
        return NodeStateDead
    default:
        return NodeStateUnknown
    }
}