# 负载均衡

此目录实现集群负载均衡：
- 负载监控和评估
  - 节点性能指标收集（CPU、内存、磁盘、网络）
  - 数据分布不均衡度计算
  - 热点数据识别
- 数据迁移策略
  - 加权得分策略：综合考虑CPU、内存、磁盘、网络因素
  - 容量均衡策略：主要考虑存储空间分布
  - 访问频率均衡策略：考虑数据访问热度
  - 复合策略：组合多种策略的优势
- 热点识别和处理
  - 访问频率统计
  - 自适应阈值检测
  - 热点分片复制与迁移
- 均衡算法实现
  - 贪心算法：从最重负载节点迁移到最轻负载节点
  - 分批迁移：控制单次迁移量，避免系统负担
  - 任务调度：优先级队列与并发控制

## 架构设计

负载均衡模块主要由以下组件组成：
1. Manager - 负载均衡管理器，控制整体流程
2. MetricCollector - 节点指标收集器
3. BalanceStrategy - 负载均衡策略接口及实现
4. Migrator - 数据迁移执行器

## 负载均衡策略

系统支持多种负载均衡策略：

1. **WeightedScoreStrategy** - 加权得分策略
   - 综合考虑CPU、内存、磁盘和分片数量
   - 适合一般场景的综合平衡

2. **CapacityBalanceStrategy** - 容量均衡策略
   - 主要关注磁盘使用率
   - 适合存储空间不平衡的场景

3. **AccessFrequencyStrategy** - 访问频率均衡策略
   - 关注热点数据和访问负载
   - 适合访问模式不均匀的场景

4. **CompositeStrategy** - 复合策略
   - 组合多种策略的优势
   - 可配置权重，灵活适应不同场景

## 使用方式

```go
// 创建并启动负载均衡管理器
manager, err := rebalance.NewManager(config, logger)
if err != nil {
    return err
}
manager.Start()

// 更新节点指标
metrics := &rebalance.NodeMetrics{...}
manager.UpdateNodeMetrics("node1", metrics)

// 手动触发负载均衡
manager.TriggerRebalance()

// 获取负载均衡状态
status := manager.GetStatus()

// 创建自定义策略
strategy := rebalance.NewWeightedScoreStrategy(0.4, 0.2, 0.2, 0.2)

// 创建复合策略
strategies := []rebalance.BalanceStrategy{
    rebalance.NewWeightedScoreStrategy(0.4, 0.2, 0.2, 0.2),
    rebalance.NewCapacityBalanceStrategy(15.0),
}
weights := []float64{0.7, 0.3}
compositeStrategy := rebalance.NewCompositeStrategy(strategies, weights)
```
