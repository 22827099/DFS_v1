package raft

import (
	etcdraft "go.etcd.io/etcd/raft/v3"
)

// Config 包含Raft节点的配置项
type Config struct {
	// 节点ID
	NodeID uint64
	// 集群成员列表
	Peers []uint64
	// 心跳超时时间(毫秒)
	HeartbeatTick int
	// 选举超时时间(毫秒)
	ElectionTick int
	// 存储目录
	StorageDir string
	// 单次快照数据大小限制
	SnapshotChunkSize uint64
	// 应用通道缓冲大小
	ApplyBufferSize int
	// 发送通道缓冲大小
	SendBufferSize int
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		NodeID:            1,
		Peers:             []uint64{1},
		HeartbeatTick:     1,
		ElectionTick:      10,
		StorageDir:        "./raft-data",
		SnapshotChunkSize: 1024 * 1024, // 1MB
		ApplyBufferSize:   1024,
		SendBufferSize:    1024,
	}
}

// ToEtcdConfig 转换为etcd/raft库的配置
func (c *Config) ToEtcdConfig() *etcdraft.Config {
	return &etcdraft.Config{
		ID:              c.NodeID,
		ElectionTick:    c.ElectionTick,
		HeartbeatTick:   c.HeartbeatTick,
		MaxSizePerMsg:   1024 * 1024,
		MaxInflightMsgs: 256,
	}
}