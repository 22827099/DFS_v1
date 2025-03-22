package rebalance

import (
	"sync"

	"github.com/22827099/DFS_v1/common/types"
)

// MetricCollector 节点指标收集器
type MetricCollector struct {
	metrics     map[string]*types.NodeMetrics // 节点ID -> 指标
	metricsLock sync.RWMutex                  // 保护metrics的互斥锁
}

// NewMetricCollector 创建新的指标收集器
func NewMetricCollector() *MetricCollector {
	return &MetricCollector{
		metrics: make(map[string]*types.NodeMetrics),
	}
}

// UpdateNodeMetrics 更新节点指标
func (c *MetricCollector) UpdateNodeMetrics(nodeID string, metrics *types.NodeMetrics) {
	c.metricsLock.Lock()
	defer c.metricsLock.Unlock()

	// 存储指标副本
	metricsCopy := *metrics
	c.metrics[nodeID] = &metricsCopy
}

// GetNodeMetrics 获取节点指标
func (c *MetricCollector) GetNodeMetrics(nodeID string) *types.NodeMetrics {
	c.metricsLock.RLock()
	defer c.metricsLock.RUnlock()

	if metrics, exists := c.metrics[nodeID]; exists {
		// 返回副本以避免并发修改
		metricsCopy := *metrics
		return &metricsCopy
	}

	return nil
}

// GetAllMetrics 获取所有节点指标
func (c *MetricCollector) GetAllMetrics() map[string]*types.NodeMetrics {
	c.metricsLock.RLock()
	defer c.metricsLock.RUnlock()

	// 创建副本
	result := make(map[string]*types.NodeMetrics, len(c.metrics))
	for nodeID, metrics := range c.metrics {
		metricsCopy := *metrics
		result[nodeID] = &metricsCopy
	}

	return result
}

// CalculateClusterStats 计算集群整体统计信息
func (mc *MetricCollector) CalculateClusterStats() *ClusterStats {
	mc.metricsLock.RLock()
	defer mc.metricsLock.RUnlock()

	if len(mc.metrics) == 0 {
		return &ClusterStats{}
	}

	stats := &ClusterStats{
		NodeCount: len(mc.metrics),
	}

	// 计算平均值和标准差
	for _, metrics := range mc.metrics {
		stats.TotalStorage += metrics.DiskCapacityBytes
		stats.UsedStorage += metrics.DiskUsageBytes
		stats.TotalShardCount += metrics.ShardCount

		stats.AvgCPUUsage += metrics.CPUUsagePercent
		stats.AvgMemoryUsage += float64(metrics.MemoryUsageBytes) / float64(1<<30) // 转换为GB
		stats.AvgDiskUsage += metrics.DiskUsageRatio * 100                         // 修改为正确字段名，转换为百分比
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
		stats.CPUStdDev += (metrics.CPUUsagePercent - stats.AvgCPUUsage) * (metrics.CPUUsagePercent - stats.AvgCPUUsage)

		memUsageGB := float64(metrics.MemoryUsageBytes) / float64(1<<30)
		stats.MemoryStdDev += (memUsageGB - stats.AvgMemoryUsage) * (memUsageGB - stats.AvgMemoryUsage)

		diskUsagePct := metrics.DiskUsageRatio * 100
		stats.DiskStdDev += (diskUsagePct - stats.AvgDiskUsage) * (diskUsagePct - stats.AvgDiskUsage)

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

