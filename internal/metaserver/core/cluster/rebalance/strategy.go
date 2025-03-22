package rebalance

import (
	"errors"
	"math"
	"sort"

	"github.com/22827099/DFS_v1/common/types"
	"github.com/google/uuid"
)

// BalanceStrategy 负载均衡策略接口
type BalanceStrategy interface {
	// Evaluate 评估集群是否需要再平衡，返回是否需要以及不平衡程度
	Evaluate(nodeMetrics map[string]*types.NodeMetrics) (bool, float64)
	// GeneratePlan 根据节点指标生成迁移计划
	GeneratePlan(nodeMetrics map[string]*types.NodeMetrics) ([]*MigrationPlan, error)
}

// MigrationPlan 数据迁移计划
type MigrationPlan struct {
	// 迁移计划ID
	PlanID string `json:"plan_id"`
	// 源节点ID
	SourceNodeID types.NodeID `json:"source_node_id"`
	// 目标节点ID
	TargetNodeID types.NodeID `json:"target_node_id"`
	// 要迁移的分片IDs
	ShardIDs []string `json:"shard_ids"`
	// 预计迁移数据量（字节）
	EstimatedBytes uint64 `json:"estimated_bytes"`
	// 优先级（1-10，值越大优先级越高）
	Priority int `json:"priority"`
}

// BaseStrategy 基础策略，提供通用功能
type BaseStrategy struct {
	// 不平衡阈值
	imbalanceThreshold float64
}

// NewBaseStrategy 创建基础策略
func NewBaseStrategy(threshold float64) *BaseStrategy {
	if threshold <= 0 {
		threshold = 20.0 // 默认20%
	}
	return &BaseStrategy{
		imbalanceThreshold: threshold,
	}
}

// WeightedScoreStrategy 加权得分策略
type WeightedScoreStrategy struct {
	*BaseStrategy
	cpuWeight    float64 // CPU使用率权重
	memoryWeight float64 // 内存使用率权重
	diskWeight   float64 // 磁盘使用率权重
	shardWeight  float64 // 分片数量权重
}

// NewWeightedScoreStrategy 创建新的加权得分策略
func NewWeightedScoreStrategy(cpuWeight, memoryWeight, diskWeight, shardWeight float64) *WeightedScoreStrategy {
	return &WeightedScoreStrategy{
		BaseStrategy: NewBaseStrategy(0),
		cpuWeight:    cpuWeight,
		memoryWeight: memoryWeight,
		diskWeight:   diskWeight,
		shardWeight:  shardWeight,
	}
}

// Evaluate 评估集群是否需要再平衡
func (s *WeightedScoreStrategy) Evaluate(nodeMetrics map[string]*types.NodeMetrics) (bool, float64) {
	if len(nodeMetrics) < 2 {
		return false, 0.0
	}

	// 计算每个节点的加权负载得分
	scores := make([]float64, 0, len(nodeMetrics))
	for _, metrics := range nodeMetrics {
		score := s.calculateNodeScore(metrics, nodeMetrics)
		scores = append(scores, score)
	}

	// 计算负载不平衡度（使用变异系数：标准差/平均值）
	var sum float64
	for _, score := range scores {
		sum += score
	}
	avg := sum / float64(len(scores))

	var squaredDiffSum float64
	for _, score := range scores {
		diff := score - avg
		squaredDiffSum += diff * diff
	}

	// 防止除零
	if avg == 0 {
		return false, 0.0
	}

	// 变异系数作为不平衡度指标
	imbalanceScore := math.Sqrt(squaredDiffSum/float64(len(scores))) / avg * 100.0

	// 如果节点数量少于3，提高阈值避免频繁迁移
	threshold := s.imbalanceThreshold
	if threshold == 0 {
		threshold = 20.0
		if len(nodeMetrics) < 3 {
			threshold = 30.0
		}
	}

	return imbalanceScore > threshold, imbalanceScore
}

// GeneratePlan 生成迁移计划
func (s *WeightedScoreStrategy) GeneratePlan(nodeMetrics map[string]*types.NodeMetrics) ([]*MigrationPlan, error) {
	if len(nodeMetrics) < 2 {
		return nil, errors.New("至少需要两个节点才能生成迁移计划")
	}

	// 计算每个节点的得分并排序
	type nodeScore struct {
		NodeID string
		Score  float64
		Metric *types.NodeMetrics
	}

	scores := make([]nodeScore, 0, len(nodeMetrics))
	for nodeID, metrics := range nodeMetrics {
		score := s.calculateNodeScore(metrics, nodeMetrics)
		scores = append(scores, nodeScore{
			NodeID: nodeID,
			Score:  score,
			Metric: metrics,
		})
	}

	// 按得分降序排序，分数越高，负载越重
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	// 找出负载最重和最轻的节点
	var plans []*MigrationPlan

	// 只考虑当前最不平衡的几对节点进行迁移
	maxPairs := int(math.Ceil(float64(len(scores)) / 3.0))
	if maxPairs < 1 {
		maxPairs = 1
	}

	// 构建从高负载节点到低负载节点的迁移计划
	for i := 0; i < maxPairs && i < len(scores)/2; i++ {
		sourceNode := scores[i]
		targetNode := scores[len(scores)-i-1]

		// 计算目标迁移量：尽量使两者负载均衡
		scoreDiff := sourceNode.Score - targetNode.Score
		if scoreDiff <= 0.1 {
			continue // 如果差异很小，不需要迁移
		}

		// 估算需要迁移的数据量
		// 这里假设分片大小相近，简单地按分片数量比例计算
		sourceShards := sourceNode.Metric.ShardCount
		targetShards := targetNode.Metric.ShardCount

		// 计算平均每个节点应有的分片数
		avgShards := float64(sourceShards+targetShards) / 2.0

		// 计算需要迁移的分片数量
		shardsToMigrate := int(math.Min(float64(sourceShards)-avgShards, avgShards-float64(targetShards)))

		// 确保至少迁移一个分片，但不超过源节点的25%
		maxMigration := sourceShards / 4
		if maxMigration < 1 {
			maxMigration = 1
		}

		if shardsToMigrate > maxMigration {
			shardsToMigrate = maxMigration
		}

		if shardsToMigrate < 1 {
			shardsToMigrate = 1
		}

		// 这里无法直接获取分片ID，所以使用占位符
		// 实际系统中需要通过存储服务获取真实的分片ID
		shardIDs := make([]string, shardsToMigrate)
		for j := 0; j < shardsToMigrate; j++ {
			shardIDs[j] = "shard_placeholder_" + sourceNode.NodeID + "_" + string(rune(j))
		}

		// 估算数据量（假设每个分片1GB大小）
		estimatedBytes := uint64(shardsToMigrate) * uint64(1024*1024*1024)

		// 创建迁移计划
		plan := &MigrationPlan{
			PlanID:         uuid.New().String(),
			SourceNodeID:   types.NodeID(sourceNode.NodeID),
			TargetNodeID:   types.NodeID(targetNode.NodeID),
			ShardIDs:       shardIDs,
			EstimatedBytes: estimatedBytes,
			Priority:       10 - i, // 优先处理负载差异最大的
		}

		plans = append(plans, plan)
	}

	return plans, nil
}

// calculateNodeScore 计算节点的加权负载得分
func (s *WeightedScoreStrategy) calculateNodeScore(metrics *types.NodeMetrics, allMetrics map[string]*types.NodeMetrics) float64 {
	// 标准化分片数量相对于集群平均水平
	avgShards := 0.0
	for _, m := range allMetrics {
		avgShards += float64(m.ShardCount)
	}
	avgShards /= float64(len(allMetrics))

	normalizedShards := 0.0
	if avgShards > 0 {
		normalizedShards = float64(metrics.ShardCount) / avgShards
	}

	// 计算加权得分
	score := s.cpuWeight*metrics.CPUUsagePercent +
		s.memoryWeight*(float64(metrics.MemoryUsageBytes)/float64(1<<30)) + // 转换为GB
		s.diskWeight*metrics.DiskUsageRatio*100 + // 转换为百分比
		s.shardWeight*normalizedShards*100.0 // 将分片比例转为0-100范围

	return score
}

// CapacityBalanceStrategy 容量均衡策略，主要关注磁盘使用率
type CapacityBalanceStrategy struct {
	*BaseStrategy
}

// NewCapacityBalanceStrategy 创建新的容量均衡策略
func NewCapacityBalanceStrategy(threshold float64) *CapacityBalanceStrategy {
	return &CapacityBalanceStrategy{
		BaseStrategy: NewBaseStrategy(threshold),
	}
}

// Evaluate 评估集群是否需要再平衡
func (s *CapacityBalanceStrategy) Evaluate(nodeMetrics map[string]*types.NodeMetrics) (bool, float64) {
	if len(nodeMetrics) < 2 {
		return false, 0.0
	}

	// 提取所有节点的磁盘使用率
	diskRatios := make([]float64, 0, len(nodeMetrics))
	for _, metric := range nodeMetrics {
		diskRatios = append(diskRatios, metric.DiskUsageRatio)
	}

	// 计算磁盘使用率的变异系数
	var sum float64
	for _, ratio := range diskRatios {
		sum += ratio
	}
	avg := sum / float64(len(diskRatios))

	var squaredDiffSum float64
	for _, ratio := range diskRatios {
		diff := ratio - avg
		squaredDiffSum += diff * diff
	}

	// 防止除零
	if avg == 0 {
		return false, 0.0
	}

	// 变异系数作为不平衡度指标
	imbalanceScore := math.Sqrt(squaredDiffSum/float64(len(diskRatios))) / avg * 100.0

	return imbalanceScore > s.imbalanceThreshold, imbalanceScore
}

// GeneratePlan 生成迁移计划
func (s *CapacityBalanceStrategy) GeneratePlan(nodeMetrics map[string]*types.NodeMetrics) ([]*MigrationPlan, error) {
	if len(nodeMetrics) < 2 {
		return nil, errors.New("至少需要两个节点才能生成迁移计划")
	}

	// 按磁盘使用率排序
	type nodeDiskUsage struct {
		NodeID    string
		DiskRatio float64
		Metric    *types.NodeMetrics
	}

	diskUsages := make([]nodeDiskUsage, 0, len(nodeMetrics))
	for nodeID, metric := range nodeMetrics {
		diskUsages = append(diskUsages, nodeDiskUsage{
			NodeID:    nodeID,
			DiskRatio: metric.DiskUsageRatio,
			Metric:    metric,
		})
	}

	// 按磁盘使用率降序排序
	sort.Slice(diskUsages, func(i, j int) bool {
		return diskUsages[i].DiskRatio > diskUsages[j].DiskRatio
	})

	var plans []*MigrationPlan
	maxPairs := int(math.Ceil(float64(len(diskUsages)) / 3.0))
	if maxPairs < 1 {
		maxPairs = 1
	}

	// 从使用率最高节点迁移到使用率最低节点
	for i := 0; i < maxPairs && i < len(diskUsages)/2; i++ {
		sourceNode := diskUsages[i]
		targetNode := diskUsages[len(diskUsages)-i-1]

		// 计算差异，如果差异小则不迁移
		diffRatio := sourceNode.DiskRatio - targetNode.DiskRatio
		if diffRatio < 0.1 { // 差异小于10%则不迁移
			continue
		}

		// 计算需要迁移的分片数量（按两节点分片总数的30%计算）
		sourceShards := sourceNode.Metric.ShardCount
		shardsToMigrate := int(float64(sourceShards) * 0.3)
		if shardsToMigrate < 1 {
			shardsToMigrate = 1
		}

		// 创建分片ID列表（占位符）
		shardIDs := make([]string, shardsToMigrate)
		for j := 0; j < shardsToMigrate; j++ {
			shardIDs[j] = "capacity_shard_" + sourceNode.NodeID + "_" + string(rune(j))
		}

		// 估算数据量
		estimatedBytes := uint64(shardsToMigrate) * uint64(1024*1024*1024) // 假设每个分片1GB

		// 创建迁移计划
		plan := &MigrationPlan{
			PlanID:         uuid.New().String(),
			SourceNodeID:   types.NodeID(sourceNode.NodeID),
			TargetNodeID:   types.NodeID(targetNode.NodeID),
			ShardIDs:       shardIDs,
			EstimatedBytes: estimatedBytes,
			Priority:       10 - i,
		}

		plans = append(plans, plan)
	}

	return plans, nil
}

// AccessFrequencyStrategy 访问频率均衡策略
type AccessFrequencyStrategy struct {
	*BaseStrategy
}

// NewAccessFrequencyStrategy 创建新的访问频率均衡策略
func NewAccessFrequencyStrategy(threshold float64) *AccessFrequencyStrategy {
	return &AccessFrequencyStrategy{
		BaseStrategy: NewBaseStrategy(threshold),
	}
}

// Evaluate 评估集群是否需要再平衡
func (s *AccessFrequencyStrategy) Evaluate(nodeMetrics map[string]*types.NodeMetrics) (bool, float64) {
	// 实现类似于其他策略，但基于访问频率指标
	// 当前NodeMetrics中还没有包含访问频率信息，这里是一个示例实现
	// 实际项目中需要扩展NodeMetrics或使用其他数据源

	// 为了示例，这里使用CPU使用率作为访问频率的替代指标
	cpuUsages := make([]float64, 0, len(nodeMetrics))
	for _, metric := range nodeMetrics {
		cpuUsages = append(cpuUsages, metric.CPUUsagePercent)
	}

	if len(cpuUsages) < 2 {
		return false, 0.0
	}

	// 计算变异系数
	var sum float64
	for _, usage := range cpuUsages {
		sum += usage
	}
	avg := sum / float64(len(cpuUsages))

	var squaredDiffSum float64
	for _, usage := range cpuUsages {
		diff := usage - avg
		squaredDiffSum += diff * diff
	}

	// 防止除零
	if avg == 0 {
		return false, 0.0
	}

	imbalanceScore := math.Sqrt(squaredDiffSum/float64(len(cpuUsages))) / avg * 100.0

	return imbalanceScore > s.imbalanceThreshold, imbalanceScore
}

// GeneratePlan 生成迁移计划
func (s *AccessFrequencyStrategy) GeneratePlan(nodeMetrics map[string]*types.NodeMetrics) ([]*MigrationPlan, error) {
	// 类似于其他策略的实现，但基于访问频率指标
	// 示例实现，使用CPU使用率作为替代

	if len(nodeMetrics) < 2 {
		return nil, errors.New("至少需要两个节点才能生成迁移计划")
	}

	// 排序节点
	type nodeCPUUsage struct {
		NodeID   string
		CPUUsage float64
		Metric   *types.NodeMetrics
	}

	cpuUsages := make([]nodeCPUUsage, 0, len(nodeMetrics))
	for nodeID, metric := range nodeMetrics {
		cpuUsages = append(cpuUsages, nodeCPUUsage{
			NodeID:   nodeID,
			CPUUsage: metric.CPUUsagePercent,
			Metric:   metric,
		})
	}

	// 降序排序
	sort.Slice(cpuUsages, func(i, j int) bool {
		return cpuUsages[i].CPUUsage > cpuUsages[j].CPUUsage
	})

	var plans []*MigrationPlan

	// 生成计划
	for i := 0; i < 2 && i < len(cpuUsages)/2; i++ {
		sourceNode := cpuUsages[i]
		targetNode := cpuUsages[len(cpuUsages)-i-1]

		// 如果差异小则不迁移
		if sourceNode.CPUUsage-targetNode.CPUUsage < 20.0 {
			continue
		}

		// 计算迁移分片数
		shardsToMigrate := sourceNode.Metric.ShardCount / 5
		if shardsToMigrate < 1 {
			shardsToMigrate = 1
		}

		// 创建分片ID列表
		shardIDs := make([]string, shardsToMigrate)
		for j := 0; j < shardsToMigrate; j++ {
			shardIDs[j] = "hotspot_shard_" + sourceNode.NodeID + "_" + string(rune(j))
		}

		// 创建迁移计划
		plan := &MigrationPlan{
			PlanID:         uuid.New().String(),
			SourceNodeID:   types.NodeID(sourceNode.NodeID),
			TargetNodeID:   types.NodeID(targetNode.NodeID),
			ShardIDs:       shardIDs,
			EstimatedBytes: uint64(shardsToMigrate) * uint64(1024*1024*1024),
			Priority:       10 - i,
		}

		plans = append(plans, plan)
	}

	return plans, nil
}

// CompositeStrategy 组合多个策略的复合策略
type CompositeStrategy struct {
	strategies []BalanceStrategy
	weights    []float64
}

// NewCompositeStrategy 创建新的复合策略
func NewCompositeStrategy(strategies []BalanceStrategy, weights []float64) *CompositeStrategy {
	// 如果权重未提供或长度不匹配，使用平均权重
	if weights == nil || len(weights) != len(strategies) {
		weights = make([]float64, len(strategies))
		for i := range weights {
			weights[i] = 1.0 / float64(len(strategies))
		}
	}

	return &CompositeStrategy{
		strategies: strategies,
		weights:    weights,
	}
}

// Evaluate 评估集群是否需要再平衡
func (s *CompositeStrategy) Evaluate(nodeMetrics map[string]*types.NodeMetrics) (bool, float64) {
	if len(s.strategies) == 0 {
		return false, 0.0
	}

	var totalScore float64
	var needRebalance bool

	// 加权计算所有策略的评估结果
	for i, strategy := range s.strategies {
		need, score := strategy.Evaluate(nodeMetrics)
		if need {
			needRebalance = true
		}
		totalScore += score * s.weights[i]
	}

	return needRebalance, totalScore
}

// GeneratePlan 生成迁移计划
func (s *CompositeStrategy) GeneratePlan(nodeMetrics map[string]*types.NodeMetrics) ([]*MigrationPlan, error) {
	if len(s.strategies) == 0 {
		return nil, errors.New("没有可用的策略")
	}

	// 使用评分最高的策略生成计划
	var bestStrategy BalanceStrategy
	var highestScore float64

	for _, strategy := range s.strategies {
		_, score := strategy.Evaluate(nodeMetrics)
		if score > highestScore {
			highestScore = score
			bestStrategy = strategy
		}
	}

	return bestStrategy.GeneratePlan(nodeMetrics)
}
