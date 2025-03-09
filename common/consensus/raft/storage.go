package raft

import (
	"sync"

	"go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
)

// RaftStorage 定义持久化存储接口
type RaftStorage interface {
	// 初始化存储
	Initialize() error
	// 保存日志条目
	SaveEntries(entries []raftpb.Entry) error
	// 保存状态
	SaveState(state raftpb.HardState) error
	// 保存快照
	SaveSnapshot(snapshot raftpb.Snapshot) error
	// 关闭存储
	Close() error
}

// MemoryRaftStorage 实现基于etcd/raft的内存存储
type MemoryRaftStorage struct {
	storage *raft.MemoryStorage
	mu      sync.Mutex
}

// NewMemoryRaftStorage 创建内存存储
func NewMemoryRaftStorage() *MemoryRaftStorage {
	return &MemoryRaftStorage{
		storage: raft.NewMemoryStorage(),
	}
}

// Initialize 初始化存储
func (s *MemoryRaftStorage) Initialize() error {
	return nil
}

// SaveEntries 保存日志条目
func (s *MemoryRaftStorage) SaveEntries(entries []raftpb.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.storage.Append(entries)
}

// SaveState 保存硬状态
func (s *MemoryRaftStorage) SaveState(state raftpb.HardState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.storage.SetHardState(state)
}

// SaveSnapshot 保存快照
func (s *MemoryRaftStorage) SaveSnapshot(snapshot raftpb.Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.storage.ApplySnapshot(snapshot)
}

// Close 关闭存储
func (s *MemoryRaftStorage) Close() error {
	return nil
}

// 获取底层的etcd/raft存储实现
func (s *MemoryRaftStorage) EtcdStorage() *raft.MemoryStorage {
	return s.storage
}
