package heartbeat

import (
	"context"
	"sync"
	"time"

	"github.com/22827099/DFS_v1/common/logging"
	httplib "github.com/22827099/DFS_v1/common/network/http"
	"github.com/22827099/DFS_v1/internal/types"
	"github.com/22827099/DFS_v1/internal/metaserver/config"
)

// StateChange 表示节点状态变化
type StateChange struct {
	NodeID string
	State  types.NodeStatus
}

// Manager 管理节点心跳检测
type Manager struct {
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	cfg           *config.HeartbeatConfig
	nodeStates    map[string]*nodeState
	stateChangeCh chan StateChange
	logger        logging.Logger
}

// nodeState 内部节点状态记录
type nodeState struct {
	NodeID        string
	State         types.NodeStatus
	LastHeartbeat time.Time
	FailCount     int
}

// NewManager 创建心跳管理器
func NewManager(cfg *config.HeartbeatConfig, logger logging.Logger) (*Manager, error) {
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = 1 * time.Second
	}
	if cfg.SuspectTimeout == 0 {
		cfg.SuspectTimeout = 3 * time.Second
	}
	if cfg.DeadTimeout == 0 {
		cfg.DeadTimeout = 10 * time.Second
	}
	if cfg.CleanupInterval == 0 {
		cfg.CleanupInterval = 30 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		cfg:           cfg,
		nodeStates:    make(map[string]*nodeState),
		stateChangeCh: make(chan StateChange, 100),
		ctx:           ctx,
		cancel:        cancel,
		logger:        logger,
	}, nil
}

// Start 启动心跳管理
func (m *Manager) Start() error {
	m.logger.Info("启动心跳检测")

	// 启动心跳发送协程
	go m.sendHeartbeats()

	// 启动心跳检查协程
	go m.checkHeartbeats()

	// 启动过期节点清理协程
	go m.cleanupDeadNodes()

	return nil
}

// Stop 停止心跳管理
func (m *Manager) Stop() error {
	m.logger.Info("停止心跳检测")
	m.cancel()
	return nil
}

// RegisterNode 注册节点进行心跳监控
func (m *Manager) RegisterNode(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nodeStates[nodeID] = &nodeState{
		NodeID:        nodeID,
		State:         types.NodeStatusHealthy,
		LastHeartbeat: time.Now(),
		FailCount:     0,
	}

	m.logger.Info("注册节点进行心跳监控", "nodeID", nodeID)
}

// UnregisterNode 取消节点的心跳监控
func (m *Manager) UnregisterNode(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.nodeStates, nodeID)
	m.logger.Info("取消节点的心跳监控", "nodeID", nodeID)
}

// RecordHeartbeat 记录收到的心跳
func (m *Manager) RecordHeartbeat(nodeID string) {	
	m.mu.Lock()
	defer m.mu.Unlock()

	if state, exists := m.nodeStates[nodeID]; exists {
		oldState := state.State
		state.LastHeartbeat = time.Now()
		state.FailCount = 0
		state.State = types.NodeStatusHealthy

		if oldState != types.NodeStatusHealthy {
			m.stateChangeCh <- StateChange{
				NodeID: nodeID,
				State:  types.NodeStatusHealthy,
			}
		}
	} else {
		// 新节点，自动注册
		m.nodeStates[nodeID] = &nodeState{
			NodeID:        nodeID,
			State:         types.NodeStatusHealthy,
			LastHeartbeat: time.Now(),
			FailCount:     0,
		}

		m.stateChangeCh <- StateChange{
			NodeID: nodeID,
			State:  types.NodeStatusHealthy,
		}
	}
}

// StateChangeChan 返回状态变化通知通道
func (m *Manager) StateChangeChan() <-chan StateChange {
	return m.stateChangeCh
}

// 发送心跳
func (m *Manager) sendHeartbeats() {
	ticker := time.NewTicker(m.cfg.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			// 向所有注册的节点发送心跳
			m.mu.RLock()
			for nodeID := range m.nodeStates {
				// 跳过自己
				if nodeID == m.cfg.NodeID {
					continue
				}
				go m.sendHeartbeatToNode(nodeID)
			}
			m.mu.RUnlock()
		}
	}
}

// 向单个节点发送心跳
func (m *Manager) sendHeartbeatToNode(nodeID string) {
    // 获取节点地址
    baseURL := m.getNodeURL(nodeID)
    
    // 创建自定义HTTP客户端
    client := httplib.NewClient(baseURL, httplib.WithTimeout(5*time.Second))
    
    m.logger.Debug("发送心跳", "to", nodeID, "from", m.cfg.NodeID, "url", baseURL)
    
    // 发送心跳请求
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // 准备心跳数据
    heartbeatData := map[string]string{
        "sender_id": m.cfg.NodeID,
        "timestamp": time.Now().Format(time.RFC3339),
    }
    
    // 发送POST请求，注意使用client实例调用PostJSON方法
    var response map[string]interface{}
    err := client.PostJSON(ctx, "/api/v1/heartbeat", heartbeatData, &response, nil)
    if err != nil {
        m.logger.Error("发送心跳失败", "to", nodeID, "error", err)
        return
    }
    
    m.logger.Debug("心跳响应", "from", nodeID, "response", response)
}

// 辅助方法：根据节点ID获取节点URL
func (m *Manager) getNodeURL(nodeID string) string {
    // 在实际实现中，应该从配置或服务发现中获取节点地址
    // 这里简单示例，实际应用需要替换
    return "http://" + nodeID + ":8080"
}

// 检查心跳状态
func (m *Manager) checkHeartbeats() {
	ticker := time.NewTicker(m.cfg.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			m.mu.Lock()

			for nodeID, state := range m.nodeStates {
				// 跳过自己
				if nodeID == m.cfg.NodeID {
					continue
				}

				timeSinceLastHeartbeat := now.Sub(state.LastHeartbeat)

				// 处理超时的节点
				if state.State == types.NodeStatusHealthy && timeSinceLastHeartbeat > m.cfg.SuspectTimeout {
					state.State = types.NodeStatusSuspect
					state.FailCount++
					m.stateChangeCh <- StateChange{
						NodeID: nodeID,
						State:  types.NodeStatusSuspect,
					}
					m.logger.Warn("节点可疑", "nodeID", nodeID, "lastHeartbeat", state.LastHeartbeat)
				} else if state.State == types.NodeStatusSuspect && timeSinceLastHeartbeat > m.cfg.DeadTimeout {
					state.State = types.NodeStatusDead
					m.stateChangeCh <- StateChange{
						NodeID: nodeID,
						State:  types.NodeStatusDead,
					}
					m.logger.Error("节点死亡", "nodeID", nodeID, "lastHeartbeat", state.LastHeartbeat)
				}
			}

			m.mu.Unlock()
		}
	}
}

// 清理长期不活跃的节点
func (m *Manager) cleanupDeadNodes() {
	ticker := time.NewTicker(m.cfg.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			m.mu.Lock()

			for nodeID, state := range m.nodeStates {
				// 删除长时间处于死亡状态的节点
				if state.State == types.NodeStatusDead && now.Sub(state.LastHeartbeat) > 3*m.cfg.DeadTimeout {
					delete(m.nodeStates, nodeID)
					m.logger.Info("清理长期不活跃的节点", "nodeID", nodeID)
				}
			}

			m.mu.Unlock()
		}
	}
}

// GetAllNodeStates 返回所有节点的状态信息
func (m *Manager) GetAllNodeStates() map[string]types.NodeStatus {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    // 创建副本以避免并发访问问题
    result := make(map[string]types.NodeStatus, len(m.nodeStates))
    for id, state := range m.nodeStates {
        result[id] = state.State
    }
    
    return result
}

// GetNodeState 返回指定节点的状态
func (m *Manager) GetNodeState(nodeID string) types.NodeStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if state, exists := m.nodeStates[nodeID]; exists {
		return state.State
	}
	
	return types.NodeStatusUnknown
}