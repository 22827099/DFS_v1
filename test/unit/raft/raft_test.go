package raft

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/22827099/DFS_v1/common/consensus/raft"
	"go.etcd.io/etcd/raft/v3/raftpb"
)

// TestTransport 实现测试用的Transport接口
type TestTransport struct {
	nodeID       uint64
	peers        map[uint64]*TestTransport
	receiveC     chan raftpb.Message
	mu           sync.RWMutex
	disconnected bool
}

// NewTestTransport 创建测试用传输层
func NewTestTransport(nodeID uint64) *TestTransport {
	return &TestTransport{
		nodeID:   nodeID,
		peers:    make(map[uint64]*TestTransport),
		receiveC: make(chan raftpb.Message, 1024),
	}
}

// Send 向其他节点发送消息
func (t *TestTransport) Send(msgs []raftpb.Message) {
	t.mu.RLock()
	disconnected := t.disconnected
	t.mu.RUnlock()

	if disconnected {
		return
	}

	for _, msg := range msgs {
		t.mu.RLock()
		peer, ok := t.peers[msg.To]
		t.mu.RUnlock()

		if ok {
			select {
			case peer.receiveC <- msg:
			default:
				// 如果通道满了，模拟网络丢包
			}
		}
	}
}

// Start 启动传输层
func (t *TestTransport) Start() error {
	return nil
}

// Stop 停止传输层
func (t *TestTransport) Stop() {
}

// AddPeer 添加一个对等节点
func (t *TestTransport) AddPeer(id uint64, peer *TestTransport) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.peers[id] = peer
}

// Disconnect 模拟节点断开连接
func (t *TestTransport) Disconnect() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.disconnected = true
}

// Reconnect 模拟节点重新连接
func (t *TestTransport) Reconnect() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.disconnected = false
}

// 测试集群
type TestCluster struct {
	nodes      map[uint64]*raft.RaftNode
	transports map[uint64]*TestTransport
}

// NewTestCluster 创建测试集群
func NewTestCluster(t *testing.T, nodeCount int) *TestCluster {
	tc := &TestCluster{
		nodes:      make(map[uint64]*raft.RaftNode),
		transports: make(map[uint64]*TestTransport),
	}

	// 构建节点ID列表
	var peerIDs []uint64
	for i := 1; i <= nodeCount; i++ {
		peerIDs = append(peerIDs, uint64(i))
	}

	// 创建传输层
	for _, id := range peerIDs {
		tc.transports[id] = NewTestTransport(id)
	}

	// 添加对等连接
	for _, srcID := range peerIDs {
		for _, dstID := range peerIDs {
			if srcID != dstID {
				tc.transports[srcID].AddPeer(dstID, tc.transports[dstID])
			}
		}
	}

	// 创建节点
	for _, id := range peerIDs {
		config := raft.DefaultConfig()
		config.NodeID = id
		config.Peers = peerIDs
		config.HeartbeatTick = 1
		config.ElectionTick = 10
		config.ApplyBufferSize = 100
		config.SendBufferSize = 100

		node, err := raft.NewRaftNode(config, tc.transports[id])
		if err != nil {
			t.Fatalf("创建节点失败: %v", err)
		}

		tc.nodes[id] = node

		// 启动接收协程
		go tc.receiveLoop(id)
	}

	return tc
}

// 接收并处理消息
func (tc *TestCluster) receiveLoop(nodeID uint64) {
	transport := tc.transports[nodeID]
	for msg := range transport.receiveC {
		node := tc.nodes[nodeID]
		// 调用公开的 Step 方法
		node.Step(context.Background(), msg)
	}
}

// Stop 停止集群
func (tc *TestCluster) Stop() {
	for _, node := range tc.nodes {
		node.Stop()
	}
}

// WaitForLeader 等待直到集群选出领导
func (tc *TestCluster) WaitForLeader(timeout time.Duration) (uint64, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for id, node := range tc.nodes {
			if node.IsLeader() {
				return id, true
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return 0, false
}

// TestBasicElection 测试基本的领导者选举
func TestBasicElection(t *testing.T) {
	cluster := NewTestCluster(t, 3)
	defer cluster.Stop()

	// 等待领导者选举完成
	leaderID, success := cluster.WaitForLeader(3 * time.Second)
	if !success {
		t.Fatal("领导者选举失败")
	}
	t.Logf("节点 %d 成为领导者", leaderID)
}

// TestLogReplication 测试日志复制
func TestLogReplication(t *testing.T) {
	cluster := NewTestCluster(t, 3)
	defer cluster.Stop()

	// 等待领导者选举完成
	leaderID, success := cluster.WaitForLeader(3 * time.Second)
	if !success {
		t.Fatal("领导者选举失败")
	}

	leader := cluster.nodes[leaderID]

	// 提交一条日志
	testData := []byte("test-command")
	if !leader.Propose(testData) {
		t.Fatal("提案失败")
	}

	// 等待日志应用
	timeout := time.After(3 * time.Second)
	var appliedCount int
	// 用于收集各节点收到正确应用的命令
	applied := make(map[uint64][]byte)
	t.Logf("期望的命令: %s", testData)

	// 只记录内容与 testData 长度一致且匹配的命令
	for appliedCount < 3 {
		select {
		case <-timeout:
			t.Fatalf("等待日志复制超时，仅有 %d 个节点应用了正确的命令", appliedCount)
		default:
			for id, node := range cluster.nodes {
				// 如果该节点还未记录正确的命令，则尝试从 ApplyCh 中获取
				if applied[id] == nil {
					select {
					case msg := <-node.ApplyCh():
						// 只统计长度与预期相符的命令
						if msg.CommandValid && len(msg.Command) == len(testData) && string(msg.Command) == string(testData) {
							t.Logf("节点 %d 应用了命令: %q", id, string(msg.Command))
							applied[id] = msg.Command
							appliedCount++
						} else {
							// 忽略那些不符合预期的消息（例如配置变更产生的消息）
							t.Logf("节点 %d 收到非预期命令: %q", id, msg.Command)
						}
					default:
						// 无消息则继续
					}
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	// 验证所有节点应用的是相同的命令
	t.Logf("验证应用命令...")
	for id, command := range applied {
		if string(command) != string(testData) {
			t.Errorf("节点 %d 应用了错误的命令: %q, 期望: %q", id, string(command), string(testData))
		} else {
			t.Logf("节点 %d 应用了正确的命令: %q", id, string(command))
		}
	}
}

// TestFaultTolerance 测试容错性
func TestFaultTolerance(t *testing.T) {
	cluster := NewTestCluster(t, 5)
	defer cluster.Stop()

	// 等待领导者选举完成
	leaderID, success := cluster.WaitForLeader(3 * time.Second)
	if !success {
		t.Fatal("领导者选举失败")
	}

	// 模拟两个节点断开连接（少于半数）
	var failedNodes []uint64
	failCount := 0
	for id := range cluster.nodes {
		if id != leaderID && failCount < 2 {
			cluster.transports[id].Disconnect()
			failedNodes = append(failedNodes, id)
			failCount++
			t.Logf("断开节点 %d 连接", id)
		}
	}

	// 发送一个新提案
	testData := []byte("fault-tolerance-test")
	if !cluster.nodes[leaderID].Propose(testData) {
		t.Fatal("提案失败")
	}

	// 等待提案被多数节点接受
	time.Sleep(1 * time.Second)

	// 检查是否仍然有领导者
	if !cluster.nodes[leaderID].IsLeader() {
		t.Fatal("领导者状态丢失")
	}

	// 重新连接故障节点
	for _, id := range failedNodes {
		cluster.transports[id].Reconnect()
		t.Logf("重新连接节点 %d", id)
	}

	// 等待日志被所有节点同步
	time.Sleep(2 * time.Second)

	// 验证所有节点最终接收到相同的日志
	applied := make(map[uint64][]byte)
	timeout := time.After(5 * time.Second)
	expectedNodes := len(cluster.nodes)
	appliedCount := 0

	// 等待所有节点应用日志
	for appliedCount < expectedNodes {
		select {
		case <-timeout:
			// 记录哪些节点未应用日志
			var notApplied []uint64
			for id := range cluster.nodes {
				if applied[id] == nil {
					notApplied = append(notApplied, id)
				}
			}
			t.Fatalf("等待日志同步超时，以下节点未应用日志: %v", notApplied)
		default:
			for id, node := range cluster.nodes {
				if applied[id] == nil {
					select {
					case msg := <-node.ApplyCh():
						if msg.CommandValid {
							applied[id] = msg.Command
							appliedCount++
							t.Logf("节点 %d 应用了命令", id)
						}
					default:
						// 没有消息，继续检查其他节点
					}
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	// 验证所有节点应用的是相同的命令
	for id, command := range applied {
		if string(command) != string(testData) {
			t.Errorf("节点 %d 应用了错误的命令: %s, 期望: %s", id, command, testData)
		}
	}

	// 特别验证之前断开的节点是否正确同步
	for _, id := range failedNodes {
		command := applied[id]
		if string(command) != string(testData) {
			t.Errorf("故障恢复节点 %d 应用了错误的命令: %s, 期望: %s", id, command, testData)
		} else {
			t.Logf("故障恢复节点 %d 成功同步并应用了正确的命令", id)
		}
	}
}

// 更多测试用例可以添加...
