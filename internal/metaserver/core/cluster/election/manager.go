package election

import (
	"context"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/22827099/DFS_v1/common/consensus/raft"
	"github.com/22827099/DFS_v1/common/logging"
	"go.etcd.io/etcd/raft/v3/raftpb"
)

// ElectionState 表示选举状态
type ElectionState string

const (
	ElectionStateFollower  ElectionState = "follower"
	ElectionStateCandidate ElectionState = "candidate"
	ElectionStateLeader    ElectionState = "leader"
)

// Config 选举管理器配置
type Config struct {
	NodeID           string
	ElectionTimeout  time.Duration
	HeartbeatTimeout time.Duration
	PeerList         []string // 添加集群节点列表
}

// Manager 管理领导选举
type Manager struct {
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	cfg            *Config
	state          ElectionState
	currentTerm    uint64
	votedFor       string
	currentLeader  string
	lastHeartbeat  time.Time
	lastElectionTime time.Time
	electionTimer  *time.Timer
	leaderChangeCh chan string
	raftNode       *raft.RaftNode
	transport      *RaftTransport
	logger         logging.Logger
	isLeader	   bool
}

// NewManager 创建选举管理器
func NewManager(cfg *Config, logger logging.Logger) (*Manager, error) {
	if cfg.ElectionTimeout == 0 {
		cfg.ElectionTimeout = 1000 * time.Millisecond
	}
	if cfg.HeartbeatTimeout == 0 {
		cfg.HeartbeatTimeout = 500 * time.Millisecond
	}

	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		cfg:            cfg,
		state:          ElectionStateFollower,
		currentTerm:    0,
		lastHeartbeat:  time.Now(),
		lastElectionTime: time.Now(), 
		ctx:            ctx,
		cancel:         cancel,
		leaderChangeCh: make(chan string, 10),
		logger:         logger,
	}

	// 创建随机选举超时
	m.resetElectionTimer()

	// 初始化传输层
	transport := NewRaftTransport(m)

	// 初始化Raft节点
	nodeID, err := strconv.ParseUint(cfg.NodeID, 10, 64)
	if err != nil {
		return nil, err
	}

	// 创建Raft配置
	raftConfig := raft.DefaultConfig()
	raftConfig.NodeID = nodeID
	raftConfig.ElectionTick = int(cfg.ElectionTimeout / (100 * time.Millisecond))
	raftConfig.HeartbeatTick = int(cfg.HeartbeatTimeout / (100 * time.Millisecond))

	// 解析并添加集群成员
	peers := make([]uint64, 0, len(cfg.PeerList))
	for _, peerStr := range cfg.PeerList {
		peerID, err := strconv.ParseUint(peerStr, 10, 64)
		if err != nil {
			logger.Error("解析节点ID失败", "peer", peerStr, "error", err)
			continue
		}
		peers = append(peers, peerID)
	}
	raftConfig.Peers = peers

	// 创建RaftNode
	m.transport = transport
	m.raftNode, err = raft.NewRaftNode(raftConfig, transport)
	if err != nil {
		cancel()
		return nil, err
	}

	return m, nil
}

// Start 启动选举管理
func (m *Manager) Start() error {
	m.logger.Info("启动领导选举")

	// 启动传输层
	if err := m.transport.Start(); err != nil {
		return err
	}

	// 启动选举状态处理
	go m.runElection()

	// 监听Raft状态变化
	go m.monitorRaftState()

	return nil
}

// Stop 停止选举管理
func (m *Manager) Stop() error {
	m.logger.Info("停止领导选举")

	// 停止选举定时器
	if m.electionTimer != nil {
		m.electionTimer.Stop()
	}

	// 停止上下文
	m.cancel()

	// 停止Raft节点
	m.raftNode.Stop()
	m.transport.Stop()

	return nil
}

// GetCurrentLeader 获取当前领导者ID
func (m *Manager) GetCurrentLeader() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentLeader
}

// IsLeader 检查当前节点是否是领导者
func (m *Manager) IsLeader() bool {
	return m.raftNode.IsLeader()
}

// TriggerElection 触发新的选举
func (m *Manager) TriggerElection() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 在Raft中不需要手动触发选举，
	// Raft库会根据选举超时自动触发选举过程
	m.logger.Info("触发新的选举")

	// 重置选举定时器可能会加速触发新的选举
	m.resetElectionTimer()
}

// LeaderChangeChan 返回领导者变更通知通道
func (m *Manager) LeaderChangeChan() <-chan string {
	return m.leaderChangeCh
}

// 监控Raft状态变化
// 修改 monitorRaftState 函数
func (m *Manager) monitorRaftState() {
    applyCh := m.raftNode.ApplyCh()
    leaderCh := m.raftNode.LeaderCh() 

    for {
        select {
        case <-m.ctx.Done():
            return
        case msg := <-applyCh:
            m.handleRaftMsg(msg)
        case isLeader := <-leaderCh:
            // 领导者状态变更，更新选举时间
            m.mu.Lock()
            oldIsLeader := m.isLeader // 假设 Manager 结构体中有 isLeader 字段
            m.isLeader = isLeader
            
            // 根据是否为领导者设置当前领导者ID
            if isLeader {
                // 如果本节点成为领导者，设置当前领导者为本节点ID
                oldLeader := m.currentLeader
                m.currentLeader = m.cfg.NodeID // 使用本节点ID
                
                if oldLeader != m.currentLeader {
                    m.lastElectionTime = time.Now()
                    m.logger.Info("节点成为领导者", "node_id", m.cfg.NodeID, "election_time", m.lastElectionTime)
                    
                    // 通知领导者变更
                    select {
                    case m.leaderChangeCh <- m.currentLeader:
                        // 成功发送
                    default:
                        m.logger.Warn("领导者变更通道已满")
                    }
                }
            } else if oldIsLeader {
                // 如果本节点失去领导者身份，但我们不知道新领导者是谁
                // 可以将 currentLeader 设为空或保持不变
                m.currentLeader = "" // 或者保持不变
                m.logger.Info("节点失去领导者身份", "node_id", m.cfg.NodeID)
            }
            
            m.mu.Unlock()
        }
    }
}

// 处理Raft消息
func (m *Manager) handleRaftMsg(msg raft.ApplyMsg) {
	if msg.CommandValid {
		// 处理普通命令
		m.logger.Info("应用Raft命令", "index", msg.CommandIndex, "term", msg.CommandTerm)
		// 根据实际需要处理命令
	} else if msg.SnapshotValid {
		// 处理快照
		m.logger.Info("应用Raft快照", "index", msg.SnapshotIndex, "term", msg.SnapshotTerm)
		// 处理快照数据
	}
}

// 运行选举循环
func (m *Manager) runElection() {
	// 保留现有代码，但实际选举由Raft库管理
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-m.electionTimer.C:
			// 在使用Raft库时，我们不需要手动启动选举过程
			// 但我们可以记录超时并重置定时器
			m.logger.Debug("选举计时器超时")
			m.resetElectionTimer()
		}
	}
}

// 转变为跟随者
func (m *Manager) becomeFollower(term uint64, leaderId string) {
	// 在使用Raft库时，不需要手动管理状态变更
	// 这个函数保留以兼容现有代码，但不做实际工作
	m.mu.Lock()
	defer m.mu.Unlock()

	oldLeader := m.currentLeader
	m.currentLeader = leaderId

	// 如果领导者变化，通知变更
	if oldLeader != leaderId {
		m.lastElectionTime = time.Now()  // 更新选举时间
		m.leaderChangeCh <- leaderId
	}

	m.logger.Info("转为跟随者状态", "term", term, "leader", leaderId)
}

// 重置选举计时器
func (m *Manager) resetElectionTimer() {
	if m.electionTimer != nil {
		m.electionTimer.Stop()
	}

	// 随机化超时以防止分裂投票
	// 使用200ms到400ms的随机值
	timeout := m.cfg.ElectionTimeout + time.Duration(rand.Int63n(int64(m.cfg.ElectionTimeout)))

	m.electionTimer = time.NewTimer(timeout)
}

// HandleRequestVote 处理投票请求
func (m *Manager) HandleRequestVote(candidateID string, term uint64) (uint64, bool) {
	// 这个方法保留给上层调用，但实际上Raft库会处理投票请求
	m.logger.Info("收到投票请求", "candidateID", candidateID, "term", term)

	// 获取当前任期
	m.mu.RLock()
	currentTerm := m.currentTerm
	m.mu.RUnlock()

	// 返回当前任期和默认投票结果
	return currentTerm, false
}

// HandleAppendEntries 处理心跳或追加条目请求
func (m *Manager) HandleAppendEntries(leaderID string, term uint64) (uint64, bool) {
	// 这个方法保留给上层调用，但实际上Raft库会处理追加条目请求
	m.logger.Debug("收到追加条目请求", "leaderID", leaderID, "term", term)

	// 更新最后心跳时间并重置选举定时器
	m.mu.Lock()
	m.lastHeartbeat = time.Now()
	m.mu.Unlock()
	m.resetElectionTimer()

	// 获取当前任期
	m.mu.RLock()
	currentTerm := m.currentTerm
	m.mu.RUnlock()

	// 返回当前任期和默认结果
	return currentTerm, true
}

// GetState 获取当前节点状态
func (m *Manager) GetState() (ElectionState, uint64) {
	isLeader := m.raftNode.IsLeader()

	m.mu.RLock()
	currentTerm := m.currentTerm
	m.mu.RUnlock()

	if isLeader {
		return ElectionStateLeader, currentTerm
	}

	// 在实际应用中，我们无法确切知道是Follower还是Candidate
	// 但大多数时间应该是Follower
	return ElectionStateFollower, currentTerm
}

// GetVotedFor 获取投票对象
func (m *Manager) GetVotedFor() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.votedFor
}

// TransferLeadership 尝试转移领导权到指定节点
func (m *Manager) TransferLeadership(targetNodeID string) bool {
	if !m.raftNode.IsLeader() {
		m.logger.Warn("非领导者节点无法转移领导权")
		return false
	}

	m.logger.Info("尝试转移领导权", "targetNodeID", targetNodeID)

	// 在实际应用中，可以通过发送特殊配置变更实现
	// 这里简化处理，只返回成功
	return true
}

// ResetPeers 重置集群节点列表
func (m *Manager) ResetPeers(peers []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("重置集群节点列表", "peers", peers)

	// 在实际应用中，应该通过Raft的ConfChange实现
	// 这里简化处理
	return nil
}

// AddPeer 添加新的集群节点
func (m *Manager) AddPeer(peerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("添加集群节点", "peerID", peerID)

	// 解析peerID为uint64
	id, err := strconv.ParseUint(peerID, 10, 64)
	if err != nil {
		return err
	}

	// 通过Raft协议添加节点
	cc := raftpb.ConfChange{
		Type:   raftpb.ConfChangeAddNode,
		NodeID: id,
	}

	// 在Raft中提议配置变更
	data, err := cc.Marshal()
	if err != nil {
		return err
	}

	// 提议配置变更
	if !m.raftNode.Propose(data) {
		return err
	}

	return nil
}

// RemovePeer 移除集群节点
func (m *Manager) RemovePeer(peerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("移除集群节点", "peerID", peerID)

	// 解析peerID为uint64
	id, err := strconv.ParseUint(peerID, 10, 64)
	if err != nil {
		return err
	}

	// 通过Raft协议移除节点
	cc := raftpb.ConfChange{
		Type:   raftpb.ConfChangeRemoveNode,
		NodeID: id,
	}

	// 在Raft中提议配置变更
	data, err := cc.Marshal()
	if err != nil {
		return err
	}

	// 提议配置变更
	if !m.raftNode.Propose(data) {
		return err
	}

	return nil
}

// RaftTransport 实现raft.Transport接口
type RaftTransport struct {
	nodeID   uint64
	manager  *Manager
	receiveC chan raftpb.Message
}

// NewRaftTransport 创建一个新的传输层
func NewRaftTransport(manager *Manager) *RaftTransport {
	nodeID, _ := strconv.ParseUint(manager.cfg.NodeID, 10, 64)
	return &RaftTransport{
		nodeID:   nodeID,
		manager:  manager,
		receiveC: make(chan raftpb.Message, 100),
	}
}

// Send 发送Raft消息到其他节点
func (t *RaftTransport) Send(messages []raftpb.Message) {
	for _, msg := range messages {
		// 这里应实现实际的网络传输逻辑
		// 在实际应用中，应该通过网络发送给目标节点
		t.manager.logger.Debug("发送消息", "to", msg.To, "type", msg.Type)
	}
}

// Start 启动传输层
func (t *RaftTransport) Start() error {
	return nil
}

// Stop 停止传输层
func (t *RaftTransport) Stop() {
	close(t.receiveC)
}

// 接收消息
func (t *RaftTransport) receiveMessage(msg raftpb.Message) {
	t.receiveC <- msg
}

// GetLastElectionTime 获取最后一次选举时间
func (m *Manager) GetLastElectionTime() time.Time {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.lastElectionTime
}
