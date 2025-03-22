package cluster

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/22827099/DFS_v1/common/types"
    "github.com/22827099/DFS_v1/common/logging"
    metaconfig "github.com/22827099/DFS_v1/internal/metaserver/config"
    "github.com/22827099/DFS_v1/internal/metaserver/core/cluster/election"
    "github.com/22827099/DFS_v1/internal/metaserver/core/cluster/heartbeat"
    "github.com/22827099/DFS_v1/internal/metaserver/core/cluster/rebalance"
)

// ClusterEvent 表示集群中发生的事件
type ClusterEvent struct {
    Type      string      // "leader_change", "node_status", "rebalance_status"
    NodeID    string
    Data      interface{}
    Timestamp time.Time
}

// 集群状态结构体
type clusterState struct {
    nodes        map[string]types.NodeStatus
    leader       string
    lastElection time.Time
    mu           sync.RWMutex
}

// Manager 集群管理器
type ClusterManager struct {
    cfg           metaconfig.ClusterConfig
    logger        logging.Logger
    electionMgr   *election.Manager
    heartbeatMgr  *heartbeat.Manager
    rebalanceMgr  *rebalance.Manager
    isLeader      bool
    nodeID        types.NodeID
    leaderChangeCh chan string
    
    // 新增状态管理
    state        clusterState
    
    // 节点缓存，减少频繁查询
    nodeCache    map[string]nodeInfoCache
    cacheMu      sync.RWMutex
    cacheTTL     time.Duration
    
    // 事件处理相关
    ctx          context.Context
    cancel       context.CancelFunc
    eventDone    chan struct{}
}

// 节点信息缓存
type nodeInfoCache struct {
    info      *types.NodeInfo
    timestamp time.Time
}

// NewManager 创建集群管理器
func NewManager(cfg metaconfig.ClusterConfig, logger logging.Logger) (Manager, error) {
    if cfg.NodeID == "" {
        return nil, fmt.Errorf("节点ID不能为空")
    }
    
    // 创建选举管理器
    electionCfg := &election.ManagerConfig{
        NodeID:           types.NodeID(cfg.NodeID),
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

    // 创建上下文，可用于取消事件循环
    ctx, cancel := context.WithCancel(context.Background())

    // 创建集群管理器
    manager := &ClusterManager{
        cfg:           cfg,
        logger:        logger.WithContext(map[string]interface{}{"component": "cluster_manager"}),
        electionMgr:   electionMgr,
        heartbeatMgr:  heartbeatMgr,
        rebalanceMgr:  rebalanceMgr,
        nodeID:        types.NodeID(cfg.NodeID),
        isLeader:      false,
        leaderChangeCh: make(chan string, 10),
        ctx:          ctx,
        cancel:       cancel,
        eventDone:    make(chan struct{}),
        state: clusterState{
            nodes: make(map[string]types.NodeStatus),
        },
        nodeCache:     make(map[string]nodeInfoCache),
        cacheTTL:      10 * time.Second, // 默认缓存10秒
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
    
    // 启动统一的事件处理循环，替代原来的多个监听goroutine
    go m.eventLoop()
    
    return nil
}

// eventLoop 统一的事件处理循环
func (m *ClusterManager) eventLoop() {
    defer close(m.eventDone)
    
    leaderCh := m.electionMgr.LeaderChangeChan()
    stateCh := m.heartbeatMgr.StateChangeChan()
    
    for {
        select {
        case <-m.ctx.Done():
            m.logger.Info("事件循环退出")
            return
            
        case leaderID, ok := <-leaderCh:
            if !ok {
                m.logger.Info("领导者变更通道已关闭")
                continue
            }
            m.handleLeaderChange(leaderID)
            
        case stateChange, ok := <-stateCh:
            if !ok {
                m.logger.Info("节点状态通道已关闭")
                continue
            }
            m.handleNodeStateChange(stateChange)
        }
    }
}

// handleLeaderChange 处理领导者变更事件
func (m *ClusterManager) handleLeaderChange(leaderID string) {
    oldIsLeader := m.isLeader
    m.isLeader = leaderID == string(m.nodeID)
    
    // 更新集群状态
    m.state.mu.Lock()
    m.state.leader = leaderID
    m.state.lastElection = time.Now()
    m.state.mu.Unlock()
    
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
    
    // 清除缓存，确保节点信息反映最新的领导者状态
    m.cacheMu.Lock()
    m.nodeCache = make(map[string]nodeInfoCache)
    m.cacheMu.Unlock()
}

// handleNodeStateChange 处理节点状态变更事件
func (m *ClusterManager) handleNodeStateChange(change heartbeat.StateChange) {
    m.logger.Info("节点状态变更", 
        "node_id", change.NodeID, 
        "state", change.State)
    
    // 更新集群状态
    m.state.mu.Lock()
    m.state.nodes[change.NodeID] = change.State
    m.state.mu.Unlock()
    
    // 清除对应节点的缓存
    m.cacheMu.Lock()
    delete(m.nodeCache, change.NodeID)
    m.cacheMu.Unlock()
    
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
        // 节点恢复健康，如果是领导者且节点不在集群中，考虑添加回集群
        if m.IsLeader() && !m.isPeerActive(change.NodeID) {
            m.logger.Info("检测到节点恢复健康，尝试添加回集群", "node_id", change.NodeID)
            if err := m.AddPeer(change.NodeID); err != nil {
                m.logger.Error("将恢复的节点添加回集群失败", "node_id", change.NodeID, "error", err)
            }
        }
    }
}

// 检查节点是否已经在活跃的集群成员中
func (m *ClusterManager) isPeerActive(nodeID string) bool {
    // TODO: 实现检查节点是否在活跃的集群成员中的逻辑
    // 这需要依赖于electionMgr提供获取当前成员列表的方法
    return false
}

// Stop 停止集群管理器
func (m *ClusterManager) Stop(ctx context.Context) error {
    m.logger.Info("停止集群管理器")
    
    // 取消事件循环
    m.cancel()
    
    // 等待事件循环退出
    select {
    case <-m.eventDone:
        // 事件循环已正常退出
    case <-ctx.Done():
        m.logger.Warn("等待事件循环退出超时")
    }
    
    // 关闭通道，避免goroutine泄漏
    close(m.leaderChangeCh)
    
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
    // 优先从缓存的状态获取领导者ID
    m.state.mu.RLock()
    leaderID := m.state.leader
    m.state.mu.RUnlock()
    
    if leaderID != "" {
        return leaderID
    }
    
    // 如果缓存中没有，则从选举管理器获取
    return m.electionMgr.GetCurrentLeader()
}

// RegisterNode 注册新的集群节点
func (m *ClusterManager) RegisterNode(nodeID string) {
    m.logger.Info("注册新节点", "node_id", nodeID)
    m.heartbeatMgr.RegisterNode(nodeID)
}

// UnregisterNode 取消注册集群节点
func (m *ClusterManager) UnregisterNode(nodeID string) {
    m.logger.Info("注销节点", "node_id", nodeID)
    m.heartbeatMgr.UnregisterNode(nodeID)
    
    // 清除该节点的缓存
    m.cacheMu.Lock()
    delete(m.nodeCache, nodeID)
    m.cacheMu.Unlock()
}

// AddPeer 添加新的集群节点到选举组
func (m *ClusterManager) AddPeer(peerID string) error {
    m.logger.Info("添加节点到集群", "peer_id", peerID)
    return m.electionMgr.AddPeer(peerID)
}

// RemovePeer 从选举组中移除集群节点
func (m *ClusterManager) RemovePeer(peerID string) error {
    m.logger.Info("从集群中移除节点", "peer_id", peerID)
    
    err := m.electionMgr.RemovePeer(peerID)
    if err != nil {
        m.logger.Error("从选举组移除节点失败", "peer_id", peerID, "error", err)
        return fmt.Errorf("移除节点失败: %w", err)
    }
    
    // 同时从心跳管理中注销节点
    m.UnregisterNode(peerID)
    return nil
}

// TriggerRebalance 手动触发负载均衡
func (m *ClusterManager) TriggerRebalance() {
    // 只有领导者节点才能触发负载均衡
    if !m.IsLeader() {
        m.logger.Warn("只有领导者节点才能触发负载均衡")
        return
    }
    
    m.logger.Info("手动触发负载均衡")
    m.rebalanceMgr.TriggerRebalance()
}

// GetRebalanceStatus 获取负载均衡状态
func (m *ClusterManager) GetRebalanceStatus() map[string]interface{} {
    return m.rebalanceMgr.GetStatus()
}

// UpdateNodeMetrics 更新节点度量指标
func (m *ClusterManager) UpdateNodeMetrics(nodeID string, metrics *types.NodeMetrics) {
    m.rebalanceMgr.UpdateNodeMetrics(nodeID, metrics)
    
    // 更新后清除该节点的缓存，确保下次获取能拿到最新指标
    m.cacheMu.Lock()
    delete(m.nodeCache, nodeID)
    m.cacheMu.Unlock()
}

// LeaderChangeChan 返回领导者变更通知通道
func (m *ClusterManager) LeaderChangeChan() <-chan string {
    return m.leaderChangeCh
}

// 成为领导者时的处理
func (m *ClusterManager) onBecomeLeader() {
    m.logger.Info("本节点成为集群领导者")
    
    // 领导者节点负责触发负载均衡等操作
    go func() {
        // 等待一段时间再触发负载均衡，给系统一些稳定时间
        select {
        case <-time.After(5 * time.Second):
            if m.IsLeader() { // 再次检查，防止在等待期间失去领导权
                m.TriggerRebalance()
            }
        case <-m.ctx.Done():
            return
        }
    }()
}

// 失去领导权时的处理
func (m *ClusterManager) onLoseLeadership() {
    m.logger.Info("本节点失去集群领导权")
    
    // 清理只有领导者才应该执行的任务
}

// ListNodes 获取当前集群所有节点信息
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
        // 尝试从缓存获取
        cachedInfo := m.getCachedNodeInfo(nodeID)
        if cachedInfo != nil {
            nodes = append(nodes, *cachedInfo)
            continue
        }
        
        // 缓存未命中，构建基本节点信息
        nodeInfo := m.buildNodeInfo(nodeID, state, leaderID)
        
        // 获取并添加节点指标数据
        m.addMetricsToNodeInfo(&nodeInfo, nodeID)
        
        // 更新缓存
        m.updateNodeInfoCache(nodeID, &nodeInfo)
        
        nodes = append(nodes, nodeInfo)
    }
    
    m.logger.Debug("获取到节点列表", "count", len(nodes))
    return nodes, nil
}

// getCachedNodeInfo 从缓存获取节点信息
func (m *ClusterManager) getCachedNodeInfo(nodeID string) *types.NodeInfo {
    m.cacheMu.RLock()
    defer m.cacheMu.RUnlock()
    
    cache, ok := m.nodeCache[nodeID]
    if ok && time.Since(cache.timestamp) < m.cacheTTL {
        // 复制一份返回，避免修改缓存
        infoCopy := *cache.info
        return &infoCopy
    }
    
    return nil
}

// updateNodeInfoCache 更新节点信息缓存
func (m *ClusterManager) updateNodeInfoCache(nodeID string, info *types.NodeInfo) {
    m.cacheMu.Lock()
    defer m.cacheMu.Unlock()
    
    // 创建一个副本存入缓存
    infoCopy := *info
    m.nodeCache[nodeID] = nodeInfoCache{
        info:      &infoCopy,
        timestamp: time.Now(),
    }
}

// buildNodeInfo 构建基本的节点信息
func (m *ClusterManager) buildNodeInfo(nodeID string, state types.NodeStatus, leaderID string) types.NodeInfo {
    // 转换节点状态为通用类型
    status := m.convertNodeStatus(state)
    
    return types.NodeInfo{
        NodeID:    types.NodeID(nodeID),
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
    // 直接复用获取到的metrics对象而不是创建新的
    nodeInfo.Metrics = metrics
}

// GetNodeInfo 获取指定节点的详细信息
func (m *ClusterManager) GetNodeInfo(ctx context.Context, nodeID string) (*types.NodeInfo, error) {
    // 检查上下文是否已取消
    if err := ctx.Err(); err != nil {
        return nil, fmt.Errorf("获取节点信息中断: %w", err)
    }
    
    // 先检查缓存
    cachedInfo := m.getCachedNodeInfo(nodeID)
    if cachedInfo != nil {
        return cachedInfo, nil
    }
    
    // 缓存未命中，执行原有逻辑
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
    
    // 更新缓存
    m.updateNodeInfoCache(nodeID, &nodeInfo)
    
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

// GetNodeCount 获取集群节点总数
func (m *ClusterManager) GetNodeCount() int {
    nodeStates := m.heartbeatMgr.GetAllNodeStates()
    return len(nodeStates)
}

// GetHealthyNodeCount 获取健康节点数量
func (m *ClusterManager) GetHealthyNodeCount() int {
    nodeStates := m.heartbeatMgr.GetAllNodeStates()
    count := 0
    for _, state := range nodeStates {
        if state == types.NodeStatusHealthy {
            count++
        }
    }
    return count
}

// LastElectionTime 获取最后一次选举时间
func (m *ClusterManager) LastElectionTime() time.Time {
    // 从状态中获取最后选举时间
    m.state.mu.RLock()
    lastElection := m.state.lastElection
    m.state.mu.RUnlock()
    
    if !lastElection.IsZero() {
        return lastElection
    }
    
    // 如果没有记录，返回当前时间
    return time.Now()
}

// GetClusterSnapshot 获取当前集群状态快照
func (m *ClusterManager) GetClusterSnapshot() map[string]interface{} {
    nodes, _ := m.ListNodes(context.Background())
    
    snapshot := map[string]interface{}{
        "total_nodes":      len(nodes),
        "healthy_nodes":    m.GetHealthyNodeCount(),
        "leader_id":        m.GetCurrentLeader(),
        "last_election":    m.LastElectionTime(),
        "rebalance_status": m.GetRebalanceStatus(),
    }
    
    return snapshot
}