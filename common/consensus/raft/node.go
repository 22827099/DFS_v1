package raft

import (
	"context"
	"sync"
	"time"

	"github.com/22827099/DFS_v1/common/logging"
	etcdraft "go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
)

// RaftNode 封装etcd/raft库，提供简化的接口
type RaftNode struct {
    mu          sync.RWMutex          // 读写锁
    isLeader    bool                  // 是否为领导者
    config      *Config               // 配置
    node        etcdraft.Node         // etcd/raft 节点
    raftStorage *MemoryStorage        // 内存存储
    transport   Transport             // 网络传输接口
    readyHandler *readyHandler        // Ready对象处理器
    applyCh     chan ApplyMsg         // 应用通道，用于接收已提交的日志条目
    leaderCh    chan bool             // 通知领导者变更
    proposeC    chan []byte           // 提案通道
    confChangeC chan raftpb.ConfChange // 配置变更通道
    commitC     chan *commit           // 提交通道
    done        chan struct{}          // 停止信号
    stopOnce    sync.Once              // 确保停止操作只执行一次
}


// ApplyMsg 表示需要应用到状态机的消息
type ApplyMsg struct {
	CommandValid bool
	Command      []byte
	CommandIndex uint64
	CommandTerm  uint64
	// 快照相关字段
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  uint64
	SnapshotIndex uint64
}

type commit struct {
	data       []byte
	applyDoneC chan<- struct{}
}

// Step 处理从网络接收到的 Raft 消息
func (rn *RaftNode) Step(ctx context.Context, msg raftpb.Message) error {
	return rn.node.Step(ctx, msg)
}

// NewRaftNode 创建一个新的Raft节点
func NewRaftNode(config *Config, transport Transport) (*RaftNode, error) {
	storage := NewMemoryStorage()

	etcdConfig := config.ToEtcdConfig()
	etcdConfig.Storage = storage

	// 初始化集群成员
	peers := make([]etcdraft.Peer, len(config.Peers))
	for i, id := range config.Peers {
		peers[i] = etcdraft.Peer{ID: id}
	}

	node := etcdraft.StartNode(etcdConfig, peers)

	rn := &RaftNode{
		config:      config,
		node:        node,
		raftStorage: storage,
		transport:   transport,
		applyCh:     make(chan ApplyMsg, config.ApplyBufferSize),
		proposeC:    make(chan []byte, config.SendBufferSize),
		confChangeC: make(chan raftpb.ConfChange),
		commitC:     make(chan *commit),
		done:        make(chan struct{}),
	}

	rn.readyHandler = newReadyHandler(rn)

	// 启动节点处理循环
	go rn.run()
	go rn.serveProposals()

	return rn, nil
}

// 处理Raft节点事件的主循环
func (rn *RaftNode) run() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rn.node.Tick()

		case rd := <-rn.node.Ready():
			rn.readyHandler.handleReady(rd)

		case <-rn.done:
			rn.node.Stop()
			return
		}
	}
}

// 处理提案的协程
func (rn *RaftNode) serveProposals() {
	for {
		select {
		case prop := <-rn.proposeC:
			rn.node.Propose(context.TODO(), prop)

		case cc := <-rn.confChangeC:
			rn.node.ProposeConfChange(context.TODO(), cc)

		case <-rn.done:
			return
		}
	}
}

// Propose 提交一个新的指令到Raft日志
func (rn *RaftNode) Propose(command []byte) bool {
	select {
	case rn.proposeC <- command:
		return true
	case <-rn.done:
		return false
	}
}

// Stop 停止Raft节点
func (rn *RaftNode) Stop() {
	rn.stopOnce.Do(func() {
		close(rn.done)
	})
}

// IsLeader 返回当前节点是否为领导者
func (rn *RaftNode) IsLeader() bool {
	rn.mu.RLock()
	defer rn.mu.RUnlock()
	return rn.isLeader
}

// ApplyCh 返回应用通道，用于接收已提交的日志条目
func (rn *RaftNode) ApplyCh() <-chan ApplyMsg {
	return rn.applyCh
}

// LeaderCh 返回领导者变更通知通道
func (rn *RaftNode) LeaderCh() <-chan bool {
    return rn.leaderCh
}

// readyHandler 处理Ready对象
type readyHandler struct {
	rn *RaftNode
}

func newReadyHandler(rn *RaftNode) *readyHandler {
	return &readyHandler{rn: rn}
}

func (rh *readyHandler) handleReady(rd etcdraft.Ready) {
    // 1. 持久化日志条目和 HardState
    if !etcdraft.IsEmptyHardState(rd.HardState) {
        rh.rn.raftStorage.mu.Lock()
        rh.rn.raftStorage.hardState = rd.HardState
        rh.rn.raftStorage.mu.Unlock()
    }
    
    if len(rd.Entries) > 0 {
        rh.rn.raftStorage.mu.Lock()
        if len(rh.rn.raftStorage.entries) == 0 {
            // 存储为空，直接使用新条目
            rh.rn.raftStorage.entries = append([]raftpb.Entry{}, rd.Entries...)
        } else {
            // 处理已有条目情况
            firstNewIdx := rd.Entries[0].Index
            firstStoreIdx := rh.rn.raftStorage.entries[0].Index
            
            // 计算在存储中的偏移
            offset := int(firstNewIdx - firstStoreIdx)
            
            if offset < 0 {
                // 新条目比存储的更早
                rh.rn.raftStorage.entries = append([]raftpb.Entry{}, rd.Entries...)
            } else if offset < len(rh.rn.raftStorage.entries) {
                // 有重叠，保留前面的条目，覆盖重叠部分，添加新条目
                rh.rn.raftStorage.entries = append(
                    rh.rn.raftStorage.entries[:offset],
                    rd.Entries...,
                )
            } else if offset == len(rh.rn.raftStorage.entries) {
                // 直接接续，没有间隙
                rh.rn.raftStorage.entries = append(rh.rn.raftStorage.entries, rd.Entries...)
            } else {
                // 有间隙，不应该发生，日志会丢失
                panic("raft log has gap")
            }
        }
        rh.rn.raftStorage.mu.Unlock()
    }
    
    // 2. 处理快照
    if !etcdraft.IsEmptySnap(rd.Snapshot) {
        rh.rn.raftStorage.mu.Lock()
        rh.rn.raftStorage.snapshot = rd.Snapshot
        // 快照可能会使旧日志条目过时，需要更新 entries 数组
        snapshotIndex := rd.Snapshot.Metadata.Index
        
        // 保留快照索引之后的条目
        newEntries := make([]raftpb.Entry, 0)
        for _, entry := range rh.rn.raftStorage.entries {
            if entry.Index > snapshotIndex {
                newEntries = append(newEntries, entry)
            }
        }
        rh.rn.raftStorage.entries = newEntries
        rh.rn.raftStorage.mu.Unlock()
        
        // 构造应用消息并发送到 applyCh
        applyMsg := ApplyMsg{
            SnapshotValid: true,
            Snapshot:      rd.Snapshot.Data,
            SnapshotTerm:  rd.Snapshot.Metadata.Term,
            SnapshotIndex: snapshotIndex,
        }
        rh.rn.applyCh <- applyMsg
    }
    
    // 3. 发送消息到其他节点
    if len(rd.Messages) > 0 {
        rh.rn.transport.Send(rd.Messages)
    }
    
    // 4. 应用已提交的条目到状态机
    for _, entry := range rd.CommittedEntries {
        if entry.Type == raftpb.EntryNormal && len(entry.Data) > 0 {
            // 打印日志帮助调试
        	logging.Info("应用命令，索引: %d，长度: %d\n", entry.Index, len(entry.Data))

			// 普通命令，应用到状态机
            applyMsg := ApplyMsg{
                CommandValid: true,
                Command:      append([]byte{}, entry.Data...),
                CommandIndex: entry.Index,
                CommandTerm:  entry.Term,
            }
            rh.rn.applyCh <- applyMsg
        } else if entry.Type == raftpb.EntryConfChange {
            // 处理配置变更
            var cc raftpb.ConfChange
            if err := cc.Unmarshal(entry.Data); err != nil {
                // 反序列化失败，记录错误并继续
                // 实际生产环境应该有日志记录
                continue
            }
            
            // 应用配置变更
            confState := rh.rn.node.ApplyConfChange(cc)
            
            // 更新存储的配置状态
            rh.rn.raftStorage.mu.Lock()
            rh.rn.raftStorage.confState = *confState
            rh.rn.raftStorage.mu.Unlock()
            
            // 通知上层应用配置变更
            applyMsg := ApplyMsg{
                CommandValid: true,
                Command:      entry.Data,
                CommandIndex: entry.Index,
                CommandTerm:  entry.Term,
            }
            rh.rn.applyCh <- applyMsg
        }
    }
    
    // 5. 处理领导者变更
    if rd.SoftState != nil {
        wasLeader := rh.rn.isLeader
        newIsLeader := rd.SoftState.RaftState == etcdraft.StateLeader
        
        // 只有状态变化时才需要更新
        if wasLeader != newIsLeader {
            rh.rn.mu.Lock()
            rh.rn.isLeader = newIsLeader
            rh.rn.mu.Unlock()
            
            // 可以在这里处理领导者变更的其他逻辑
            // 如：领导者选举后的初始化工作
        }
    }
    
    // 6. 通知 raft 库已处理完 Ready
    rh.rn.node.Advance()
}

// MemoryStorage 是一个内存存储实现
type MemoryStorage struct {
    // 添加必要的字段
    mu       sync.RWMutex
    hardState raftpb.HardState
    confState raftpb.ConfState
    entries  []raftpb.Entry
    snapshot raftpb.Snapshot
}

// Entries implements raft.Storage.
func (m *MemoryStorage) Entries(lo uint64, hi uint64, maxSize uint64) ([]raftpb.Entry, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    if len(m.entries) == 0 {
        return nil, etcdraft.ErrUnavailable
    }
    
    offset := m.entries[0].Index
    if lo < offset {
        return nil, etcdraft.ErrCompacted
    }
    
    if hi > offset + uint64(len(m.entries)) {
        hi = offset + uint64(len(m.entries))
    }
    
    // 计算索引
    loIdx := lo - offset
    hiIdx := hi - offset
    
    result := make([]raftpb.Entry, hiIdx-loIdx)
    copy(result, m.entries[loIdx:hiIdx])
    
    // 检查条目大小是否超过限制
    var size uint64
    for i := range result {
        size += uint64(len(result[i].Data))
        if size > maxSize && i > 0 {
            return result[:i], nil
        }
    }
    
    return result, nil
}

// FirstIndex implements raft.Storage.
func (m *MemoryStorage) FirstIndex() (uint64, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    if len(m.entries) == 0 {
        // 如果没有条目，返回快照索引+1
        return m.snapshot.Metadata.Index + 1, nil
    }
    
    return m.entries[0].Index, nil
}

// InitialState implements raft.Storage.
func (m *MemoryStorage) InitialState() (raftpb.HardState, raftpb.ConfState, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    return m.hardState, m.confState, nil
}

// LastIndex implements raft.Storage.
func (m *MemoryStorage) LastIndex() (uint64, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    if len(m.entries) == 0 {
        return m.snapshot.Metadata.Index, nil
    }
    
    return m.entries[len(m.entries)-1].Index, nil
}

// Snapshot implements raft.Storage.
func (m *MemoryStorage) Snapshot() (raftpb.Snapshot, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    if m.snapshot.Metadata.Index == 0 {
        return raftpb.Snapshot{}, etcdraft.ErrSnapshotTemporarilyUnavailable
    }
    
    return m.snapshot, nil
}

// Term implements raft.Storage.
func (m *MemoryStorage) Term(i uint64) (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Check if the requested index is in snapshot
	if i < m.snapshot.Metadata.Index {
		return 0, etcdraft.ErrCompacted
	}
	
	if len(m.entries) == 0 {
		// If there are no entries but the index matches the snapshot index
		if i == m.snapshot.Metadata.Index {
			return m.snapshot.Metadata.Term, nil
		}
		return 0, etcdraft.ErrUnavailable
	}
	
	// Calculate the relative position in entries slice
	offset := m.entries[0].Index
	if i < offset {
		return 0, etcdraft.ErrCompacted
	}
	
	if i > m.entries[len(m.entries)-1].Index {
		return 0, etcdraft.ErrUnavailable
	}
	
	return m.entries[i-offset].Term, nil
}

// NewMemoryStorage 创建一个新的内存存储
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{}
}

// Transport 定义网络传输接口
type Transport interface {
	// 发送消息到指定节点
	Send(messages []raftpb.Message)
	// 启动接收消息
	Start() error
	// 停止接收消息
	Stop()
}
