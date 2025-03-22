package types

// NodeMetrics 表示节点的性能和负载指标
type NodeMetrics struct {
	NodeID            NodeID  `json:"node_id"`             // 节点ID
	DiskUsageBytes    uint64  `json:"disk_usage_bytes"`    // 磁盘使用量（字节）
	DiskCapacityBytes uint64  `json:"disk_capacity_bytes"` // 磁盘总容量（字节）
	DiskUsageRatio    float64 `json:"disk_usage_ratio"`    // 磁盘使用率（0-1）
	CPUUsagePercent   float64 `json:"cpu_usage_percent"`   // CPU使用率（百分比）
	MemoryUsageBytes  uint64  `json:"memory_usage_bytes"`  // 内存使用量（字节）
	NetworkInBps      uint64  `json:"network_in_bps"`      // 网络入流量（字节/秒）
	NetworkOutBps     uint64  `json:"network_out_bps"`     // 网络出流量（字节/秒）
	ShardCount        int     `json:"shard_count"`         // 分片数量
	LoadScore         float64 `json:"load_score"`          // 综合负载分数
	IsHealthy         bool    `json:"is_healthy"`          // 节点是否健康
	LastUpdated       int64   `json:"last_updated"`        // 最后更新时间戳
}

// CalculateUsageRatio 计算并更新磁盘使用率
func (m *NodeMetrics) CalculateUsageRatio() float64 {
	if m.DiskCapacityBytes == 0 {
		m.DiskUsageRatio = 0
		return 0
	}
	m.DiskUsageRatio = float64(m.DiskUsageBytes) / float64(m.DiskCapacityBytes)
	return m.DiskUsageRatio
}

// CalculateLoadScore 计算综合负载分数
func (m *NodeMetrics) CalculateLoadScore() float64 {
	// 可自定义权重配置
	diskWeight := 0.7
	cpuWeight := 0.3

	diskScore := m.CalculateUsageRatio() * 100
	m.LoadScore = diskScore*diskWeight + m.CPUUsagePercent*cpuWeight
	return m.LoadScore
}
