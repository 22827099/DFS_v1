package rebalance

import (
	"errors"
	"math"
	"sort"

	"github.com/22827099/DFS_v1/internal/types"
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
	SourceNodeID string `json:"source_node_id"`
	// 目标节点ID
	TargetNodeID string `json:"target_node_id"`
	// 要迁移的分片IDs
	ShardIDs []string `json:"shard_ids"`
	// 预计迁移数据量（字节）
	EstimatedBytes uint64 `json:"estimated_bytes"`
	// 优先级（1-10，值越大优先级越高）
	Priority int `json:"priority"`
}

// WeightedScoreStrategy 加权得分策略
type WeightedScoreStrategy struct {
	cpuWeight    float64 // CPU使用率权重
	memoryWeight float64 // 内存使用率权重
	diskWeight   float64 // 磁盘使用率权重
	shardWeight  float64 // 分片数量权重
}

// NewWeightedScoreStrategy 创建新的加权得分策略
func NewWeightedScoreStrategy(cpuWeight, memoryWeight, diskWeight, shardWeight float64) *WeightedScoreStrategy {
	return &WeightedScoreStrategy{
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
	threshold := 20.0
	if len(nodeMetrics) < 3 {
		threshold = 30.0
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
			PlanID:         "plan_" + sourceNode.NodeID + "_to_" + targetNode.NodeID,
			SourceNodeID:   sourceNode.NodeID,
			TargetNodeID:   targetNode.NodeID,
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
	score := s.cpuWeight*metrics.CPUUsage +
		s.memoryWeight*metrics.MemoryUsage +
		s.diskWeight*metrics.DiskUsage +
		s.shardWeight*normalizedShards*100.0 // 将分片比例转为0-100范围

	return score
}
