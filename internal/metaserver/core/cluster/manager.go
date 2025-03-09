package cluster

import (
	"context"
	"fmt"
	"time"

	"github.com/22827099/DFS_v1/common/logging"
	metaconfig "github.com/22827099/DFS_v1/internal/metaserver/config"
	"github.com/22827099/DFS_v1/internal/metaserver/core/cluster/election"
	"github.com/22827099/DFS_v1/internal/metaserver/core/cluster/heartbeat"
	"github.com/22827099/DFS_v1/internal/metaserver/core/cluster/rebalance"
	"github.com/22827099/DFS_v1/internal/types"
)

// Manager 集群管理器
type ClusterManager struct {
	cfg           metaconfig.ClusterConfig
	logger        logging.Logger
	electionMgr   *election.Manager
	heartbeatMgr  *heartbeat.Manager
	rebalanceMgr  *rebalance.Manager
	isLeader      bool
	nodeID        string
	leaderChangeCh chan string
}

// NewManager 创建集群管理器
func NewManager(cfg metaconfig.ClusterConfig, logger logging.Logger) (Manager, error) {
	if cfg.NodeID == "" {
		return nil, fmt.Errorf("节点ID不能为空")
	}
	
	// 创建选举管理器
	electionCfg := &election.Config{
		NodeID:           cfg.NodeID,
		ElectionTimeout:  cfg.ElectionTimeout,
		HeartbeatTimeout: cfg.HeartbeatTimeout,
		PeerList:         cfg.Peers,
	}
	
	electionMgr, err := election.NewManager(electionCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("创建选举管理器失败: %w", err)
	}
	
	// 创建心跳管理器
	heartbeatCfg := &metaconfig.HeartbeatConfig{
		NodeID:            cfg.NodeID,
		HeartbeatInterval: cfg.HeartbeatInterval,
		SuspectTimeout:    cfg.SuspectTimeout,
		DeadTimeout:       cfg.DeadTimeout,
	}
	
	heartbeatMgr, err := heartbeat.NewManager(heartbeatCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("创建心跳管理器失败: %w", err)
	}
	
	// 创建负载均衡管理器
	rebalanceCfg := &metaconfig.LoadBalancerConfig{
		EvaluationInterval:      cfg.RebalanceEvaluationInterval,
		ImbalanceThreshold:      cfg.ImbalanceThreshold,
		MaxConcurrentMigrations: cfg.MaxConcurrentMigrations,
		MinMigrationInterval:    cfg.MinMigrationInterval,
		MigrationTimeout:        cfg.MigrationTimeout,
	}
	
	rebalanceMgr, err := rebalance.NewManager(rebalanceCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("创建负载均衡管理器失败: %w", err)
	}

	// 创建集群管理器
	manager := &ClusterManager{
		cfg:           cfg,
		logger:        logger.WithContext(map[string]interface{}{"component": "cluster_manager"}),
		electionMgr:   electionMgr,
		heartbeatMgr:  heartbeatMgr,
		rebalanceMgr:  rebalanceMgr,
		nodeID:        cfg.NodeID,
		isLeader:      false,
		leaderChangeCh: make(chan string, 10),
	}
	
	return manager, nil
}

// Start 启动集群管理器
func (m *ClusterManager) Start() error {
	m.logger.Info("启动集群管理器")
	
	// 启动心跳管理器
	if err := m.heartbeatMgr.Start(); err != nil {
		return fmt.Errorf("启动心跳管理器失败: %w", err)
	}
	
	// 启动选举管理器
	if err := m.electionMgr.Start(); err != nil {
		m.heartbeatMgr.Stop()
		return fmt.Errorf("启动选举管理器失败: %w", err)
	}
	
	// 启动负载均衡管理器
	if err := m.rebalanceMgr.Start(); err != nil {
		m.electionMgr.Stop()
		m.heartbeatMgr.Stop()
		return fmt.Errorf("启动负载均衡管理器失败: %w", err)
	}
	
	// 监听领导者变更事件
	go m.monitorLeaderChanges()
	
	// 监听节点状态变更
	go m.monitorNodeStateChanges()
	
	return nil
}

// Stop 停止集群管理器
func (m *ClusterManager) Stop(ctx context.Context) error {
	m.logger.Info("停止集群管理器")
	
	// 按照依赖关系的逆序停止
	var errs []error
	
	if err := m.rebalanceMgr.Stop(); err != nil {
		errs = append(errs, fmt.Errorf("停止负载均衡管理器失败: %w", err))
	}
	
	if err := m.electionMgr.Stop(); err != nil {
		errs = append(errs, fmt.Errorf("停止选举管理器失败: %w", err))
	}
	
	if err := m.heartbeatMgr.Stop(); err != nil {
		errs = append(errs, fmt.Errorf("停止心跳管理器失败: %w", err))
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("停止集群管理器时发生错误: %v", errs)
	}
	
	return nil
}

// IsLeader 检查当前节点是否为领导者
func (m *ClusterManager) IsLeader() bool {
	return m.electionMgr.IsLeader()
}

// GetCurrentLeader 获取当前领导者节点ID
func (m *ClusterManager) GetCurrentLeader() string {
	return m.electionMgr.GetCurrentLeader()
}

// RegisterNode 注册新的集群节点
func (m *ClusterManager) RegisterNode(nodeID string) {
	m.heartbeatMgr.RegisterNode(nodeID)
}

// UnregisterNode 取消注册集群节点
func (m *ClusterManager) UnregisterNode(nodeID string) {
	m.heartbeatMgr.UnregisterNode(nodeID)
}

// AddPeer 添加新的集群节点到选举组
func (m *ClusterManager) AddPeer(peerID string) error {
	return m.electionMgr.AddPeer(peerID)
}

// RemovePeer 从选举组中移除集群节点
func (m *ClusterManager) RemovePeer(peerID string) error {
	return m.electionMgr.RemovePeer(peerID)
}

// TriggerRebalance 手动触发负载均衡
func (m *ClusterManager) TriggerRebalance() {
	// 只有领导者节点才能触发负载均衡
	if !m.IsLeader() {
		m.logger.Warn("只有领导者节点才能触发负载均衡")
		return
	}
	
	m.rebalanceMgr.TriggerRebalance()
}

// GetRebalanceStatus 获取负载均衡状态
func (m *ClusterManager) GetRebalanceStatus() map[string]interface{} {
	return m.rebalanceMgr.GetStatus()
}

// UpdateNodeMetrics 更新节点度量指标
func (m *ClusterManager) UpdateNodeMetrics(nodeID string, metrics *types.NodeMetrics) {
	m.rebalanceMgr.UpdateNodeMetrics(nodeID, metrics)
}

// LeaderChangeChan 返回领导者变更通知通道
func (m *ClusterManager) LeaderChangeChan() <-chan string {
	return m.leaderChangeCh
}

// 监听领导者变更事件
func (m *ClusterManager) monitorLeaderChanges() {
	leaderCh := m.electionMgr.LeaderChangeChan()
	
	for leaderID := range leaderCh {
		oldIsLeader := m.isLeader
		m.isLeader = leaderID == m.nodeID
		
		// 记录领导者变更事件
		m.logger.Info("集群领导者变更", 
			"leader_id", leaderID, 
			"is_leader", m.isLeader)
			
		// 转发领导者变更事件到外部通道
		select {
		case m.leaderChangeCh <- leaderID:
			// 成功发送
		default:
			// 通道已满，记录警告
			m.logger.Warn("领导者变更通道已满，消息丢弃")
		}
		
		// 如果本节点成为新领导者
		if !oldIsLeader && m.isLeader {
			m.onBecomeLeader()
		}
		// 如果本节点失去领导权
		if oldIsLeader && !m.isLeader {
			m.onLoseLeadership()
		}
	}
}

// 监听节点状态变更
func (m *ClusterManager) monitorNodeStateChanges() {
	stateCh := m.heartbeatMgr.StateChangeChan()
	
	for change := range stateCh {
		m.logger.Info("节点状态变更", 
			"node_id", change.NodeID, 
			"state", change.State)
		
		// 对节点状态变更做出反应
		switch change.State {
		case types.NodeStatusDead:
			// 节点死亡，如果是集群成员则移除
			if m.IsLeader() {
				m.logger.Info("检测到节点死亡，尝试从集群中移除", "node_id", change.NodeID)
				if err := m.RemovePeer(change.NodeID); err != nil {
					m.logger.Error("从集群中移除死亡节点失败", "node_id", change.NodeID, "error", err)
				}
			}
		case types.NodeStatusHealthy:
			// 节点恢复健康，可以考虑添加回集群
		}
	}
}

// 成为领导者时的处理
func (m *ClusterManager) onBecomeLeader() {
	m.logger.Info("本节点成为集群领导者")
	
	// 领导者节点负责触发负载均衡等操作
}

// 失去领导权时的处理
func (m *ClusterManager) onLoseLeadership() {
	m.logger.Info("本节点失去集群领导权")
	
	// 清理只有领导者才应该执行的任务
}

// ListNodes 获取当前集群所有节点信息
// 该函数返回集群中所有已注册节点的详细信息，包括状态、是否为领导者、
// 最后一次活跃时间以及性能指标数据
func (m *ClusterManager) ListNodes(ctx context.Context) ([]types.NodeInfo, error) {
    m.logger.Info("获取集群节点列表")
    
    // 检查上下文是否已取消
    if err := ctx.Err(); err != nil {
        return nil, fmt.Errorf("获取节点列表中断: %w", err)
    }
    
    // 获取心跳管理器中的节点状态
    nodeStates := m.heartbeatMgr.GetAllNodeStates()
    
    // 当前领导者ID
    leaderID := m.GetCurrentLeader()
    m.logger.Debug("当前集群领导者", "leader_id", leaderID)
    
    // 构建返回结果
    nodes := make([]types.NodeInfo, 0, len(nodeStates))
    
    // 遍历所有节点
    for nodeID, state := range nodeStates {
        // 构建基本节点信息
        nodeInfo := m.buildNodeInfo(nodeID, state, leaderID)
        
        // 获取并添加节点指标数据
        m.addMetricsToNodeInfo(&nodeInfo, nodeID)
        
        nodes = append(nodes, nodeInfo)
    }
    
    m.logger.Debug("获取到节点列表", "count", len(nodes))
    return nodes, nil
}

// buildNodeInfo 构建基本的节点信息
func (m *ClusterManager) buildNodeInfo(nodeID string, state types.NodeStatus, leaderID string) types.NodeInfo {
    // 转换节点状态为通用类型
    status := m.convertNodeStatus(state)
    
    return types.NodeInfo{
        ID:    nodeID,
        Status:    status,
        IsLeader:  nodeID == leaderID,
        LastSeen:  time.Now().Unix(),
        Address:   nodeID,
    }
}

// convertNodeStatus 将心跳状态转换为通用节点状态
func (m *ClusterManager) convertNodeStatus(status types.NodeStatus) types.NodeStatus {
    return status
}

// addMetricsToNodeInfo 向节点信息中添加性能指标数据
func (m *ClusterManager) addMetricsToNodeInfo(nodeInfo *types.NodeInfo, nodeID string) {
    metrics := m.rebalanceMgr.GetNodeMetrics(nodeID)
    if metrics == nil {
        return
    }   
    nodeInfo.Metrics = &types.NodeMetrics{
        // 使用正确的字段映射
        NodeID:            nodeID,
        CPUUsage:          metrics.CPUUsage,
        MemoryUsage:       metrics.MemoryUsage,
        DiskUsage:         metrics.DiskUsage,
        NetworkThroughput: metrics.NetworkThroughput,
        TotalStorage:      metrics.TotalStorage,
        UsedStorage:       metrics.UsedStorage,
        FreeStorage:       metrics.FreeStorage,
        IOPS:              metrics.IOPS,
        ShardCount:        metrics.ShardCount,
        HotShardCount:     metrics.HotShardCount,
        UpdateTime:        metrics.UpdateTime,
    }
}

// GetNodeInfo 获取指定节点的详细信息
func (m *ClusterManager) GetNodeInfo(ctx context.Context, nodeID string) (*types.NodeInfo, error) {
    // 检查上下文是否已取消
    if err := ctx.Err(); err != nil {
        return nil, fmt.Errorf("获取节点信息中断: %w", err)
    }
    
    // 获取领导者ID
    leaderID := m.GetCurrentLeader()
    
    // 从心跳管理器获取节点状态
    nodeStatus := m.heartbeatMgr.GetNodeState(nodeID)
    if nodeStatus == types.NodeStatusUnknown {
        return nil, fmt.Errorf("节点 %s 不存在或未注册", nodeID)
    }
    
    // 构建基本节点信息
    nodeInfo := m.buildNodeInfo(nodeID, nodeStatus, leaderID)
    
    // 获取并添加节点指标数据
    m.addMetricsToNodeInfo(&nodeInfo, nodeID)
    
    return &nodeInfo, nil
}

// GetLeader 获取当前集群领导者的详细信息
func (m *ClusterManager) GetLeader(ctx context.Context) (*types.NodeInfo, error) {
    // 获取当前领导者ID
    leaderID := m.GetCurrentLeader()
    if leaderID == "" {
        return nil, fmt.Errorf("集群当前没有领导者")
    }
    
    // 复用GetNodeInfo方法获取领导者详细信息
    return m.GetNodeInfo(ctx, leaderID)
}