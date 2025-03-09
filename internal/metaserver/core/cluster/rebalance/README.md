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
```
