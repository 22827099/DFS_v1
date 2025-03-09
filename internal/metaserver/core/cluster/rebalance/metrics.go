package rebalance

import (
	"sync"
	"time"

	"github.com/22827099/DFS_v1/internal/types"
)

// MetricCollector 指标收集器
type MetricCollector struct {
	mu      sync.RWMutex
	metrics map[string]*types.NodeMetrics
}

// NewMetricCollector 创建新的指标收集器
func NewMetricCollector() *MetricCollector {
	return &MetricCollector{
		metrics: make(map[string]*types.NodeMetrics),
	}
}

// UpdateNodeMetrics 更新节点指标
func (mc *MetricCollector) UpdateNodeMetrics(nodeID string, metrics *types.NodeMetrics) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if metrics == nil {
		return
	}

	// 确保NodeID与参数一致，并更新时间戳
	metrics.NodeID = nodeID
	metrics.UpdateTime = time.Now()
	mc.metrics[nodeID] = metrics
}

// GetNodeMetrics 获取特定节点的指标
func (mc *MetricCollector) GetNodeMetrics(nodeID string) *types.NodeMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return mc.metrics[nodeID]
}

// GetAllMetrics 获取所有节点的指标
func (mc *MetricCollector) GetAllMetrics() map[string]*types.NodeMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// 创建副本以避免数据竞争
	result := make(map[string]*types.NodeMetrics, len(mc.metrics))
	for nodeID, metrics := range mc.metrics {
		// 深复制每个指标对象
		metricsCopy := *metrics
		result[nodeID] = &metricsCopy
	}

	return result
}

// CalculateClusterStats 计算集群整体统计信息
func (mc *MetricCollector) CalculateClusterStats() *ClusterStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if len(mc.metrics) == 0 {
		return &ClusterStats{}
	}

	stats := &ClusterStats{
		NodeCount: len(mc.metrics),
	}

	// 计算平均值和标准差
	for _, metrics := range mc.metrics {
		stats.TotalStorage += metrics.TotalStorage
		stats.UsedStorage += metrics.UsedStorage
		stats.TotalShardCount += metrics.ShardCount

		stats.AvgCPUUsage += metrics.CPUUsage
		stats.AvgMemoryUsage += metrics.MemoryUsage
		stats.AvgDiskUsage += metrics.DiskUsage
	}

	// 避免除零错误
	if stats.NodeCount > 0 {
		stats.AvgCPUUsage /= float64(stats.NodeCount)
		stats.AvgMemoryUsage /= float64(stats.NodeCount)
		stats.AvgDiskUsage /= float64(stats.NodeCount)
		stats.AvgShardCount = float64(stats.TotalShardCount) / float64(stats.NodeCount)
	}

	// 计算标准差
	for _, metrics := range mc.metrics {
		stats.CPUStdDev += (metrics.CPUUsage - stats.AvgCPUUsage) * (metrics.CPUUsage - stats.AvgCPUUsage)
		stats.MemoryStdDev += (metrics.MemoryUsage - stats.AvgMemoryUsage) * (metrics.MemoryUsage - stats.AvgMemoryUsage)
		stats.DiskStdDev += (metrics.DiskUsage - stats.AvgDiskUsage) * (metrics.DiskUsage - stats.AvgDiskUsage)

		shardDiff := float64(metrics.ShardCount) - stats.AvgShardCount
		stats.ShardStdDev += shardDiff * shardDiff
	}

	// 避免除零错误
	if stats.NodeCount > 1 {
		stats.CPUStdDev = stats.CPUStdDev / float64(stats.NodeCount-1)
		stats.MemoryStdDev = stats.MemoryStdDev / float64(stats.NodeCount-1)
		stats.DiskStdDev = stats.DiskStdDev / float64(stats.NodeCount-1)
		stats.ShardStdDev = stats.ShardStdDev / float64(stats.NodeCount-1)
	}

	return stats
}

// ClusterStats 集群统计信息
type ClusterStats struct {
	NodeCount       int     `json:"node_count"`
	TotalStorage    uint64  `json:"total_storage"`
	UsedStorage     uint64  `json:"used_storage"`
	TotalShardCount int     `json:"total_shard_count"`
	AvgCPUUsage     float64 `json:"avg_cpu_usage"`
	AvgMemoryUsage  float64 `json:"avg_memory_usage"`
	AvgDiskUsage    float64 `json:"avg_disk_usage"`
	AvgShardCount   float64 `json:"avg_shard_count"`
	CPUStdDev       float64 `json:"cpu_std_dev"`
	MemoryStdDev    float64 `json:"memory_std_dev"`
	DiskStdDev      float64 `json:"disk_std_dev"`
	ShardStdDev     float64 `json:"shard_std_dev"`
}
