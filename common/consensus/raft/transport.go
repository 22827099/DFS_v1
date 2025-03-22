package raft

import (
	"fmt"
	"log"
	"time"

	"go.etcd.io/etcd/raft/v3/raftpb"
)

// 简单的演示如何使用该库
func ExampleUsage() {	
	// 创建配置
	config := DefaultConfig()
	config.NodeID = 1
	config.Peers = []uint64{1, 2, 3} // 三节点集群

	// 创建简单的传输层实现
	transport := NewSimpleTransport(config.NodeID)

	// 创建Raft节点
	node, err := NewRaftNode(config, transport)
	if err != nil {
		log.Fatalf("Failed to create raft node: %v", err)
	}

	// 启动网络传输
	if err := transport.Start(); err != nil {
		log.Fatalf("Failed to start transport: %v", err)
	}

	// 应用已提交的日志
	go func() {
		for msg := range node.ApplyCh() {
			if msg.CommandValid {
				fmt.Printf("Applied command: %s at index %d\n", string(msg.Command), msg.CommandIndex)
			} else if msg.SnapshotValid {
				fmt.Printf("Applied snapshot at index %d\n", msg.SnapshotIndex)
			}
		}
	}()

	// 提交一些指令
	if node.IsLeader() {
		node.Propose([]byte("hello world"))
		node.Propose([]byte("consensus example"))
	}

	// 运行一段时间后停止
	time.Sleep(10 * time.Second)
	node.Stop()
	transport.Stop()
}

// SimpleTransport 简单传输层实现
type SimpleTransport struct {
	nodeID uint64
	// 省略具体实现...
}

// NewSimpleTransport 创建简单传输层
func NewSimpleTransport(nodeID uint64) *SimpleTransport {
	return &SimpleTransport{
		nodeID: nodeID,
	}
}

// Send 发送消息
func (t *SimpleTransport) Send(msgs []raftpb.Message) {
	// 简化实现，实际应通过网络发送
}

// Start 启动传输层
func (t *SimpleTransport) Start() error {
	return nil
}

// Stop 停止传输层
func (t *SimpleTransport) Stop() {
	// 简化实现
}
