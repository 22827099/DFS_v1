package rebalance

import (
	"context"
	"sync"
	"time"

	"github.com/22827099/DFS_v1/common/logging"
    metaconfig "github.com/22827099/DFS_v1/internal/metaserver/config"
    "github.com/22827099/DFS_v1/internal/types"
)

// Manager 负载均衡管理器
type Manager struct {
    mu              sync.RWMutex
    ctx             context.Context
    cancel          context.CancelFunc
    cfg             *metaconfig.LoadBalancerConfig
    logger          logging.Logger
    metricCollector *MetricCollector
    strategy        BalanceStrategy
    migrator        *Migrator
    lastRebalance   time.Time
    isRebalancing   bool
    triggerCh       chan struct{}
    nodeMetrics     map[string]*types.NodeMetrics     // 所有节点的性能指标
    metricsLock     sync.RWMutex                // 保护metrics的互斥锁
}

// NewManager 创建负载均衡管理器
func NewManager(cfg *metaconfig.LoadBalancerConfig, logger logging.Logger) (*Manager, error) {
    if cfg == nil {
        cfg = &metaconfig.LoadBalancerConfig{
            EvaluationInterval:      5 * time.Minute,
            ImbalanceThreshold:      20.0, // 20%
            MaxConcurrentMigrations: 5,
            MinMigrationInterval:    30 * time.Minute,
            MigrationTimeout:        2 * time.Hour,
        }
    }

    ctx, cancel := context.WithCancel(context.Background())
    
    // 创建指标收集器
    metricCollector := NewMetricCollector()
    
    // 创建默认的均衡策略
    strategy := NewWeightedScoreStrategy(0.4, 0.2, 0.2, 0.2)
    
    // 创建迁移器
    migrator := NewMigrator(ctx, cfg.MaxConcurrentMigrations, logger)

    return &Manager{
        ctx:             ctx,
        cancel:          cancel,
        cfg:             cfg,
        logger:          logger.WithContext(map[string]interface{}{"component": "rebalance"}),
        metricCollector: metricCollector,
        strategy:        strategy,
        migrator:        migrator,
        lastRebalance:   time.Time{},
        isRebalancing:   false,
        triggerCh:       make(chan struct{}, 1),
    }, nil
}

// Start 启动负载均衡管理器
func (m *Manager) Start() error {
    m.logger.Info("启动负载均衡管理器")
    
    // 启动迁移器
    m.migrator.Start()
    
    // 启动周期性评估与再平衡
    go m.runEvaluationLoop()
    
    return nil
}

// Stop 停止负载均衡管理器
func (m *Manager) Stop() error {
    m.logger.Info("停止负载均衡管理器")
    m.cancel()
    
    // 停止迁移器
    m.migrator.Stop()
    
    return nil
}

// TriggerRebalance 手动触发负载均衡
func (m *Manager) TriggerRebalance() {
    m.logger.Info("手动触发负载均衡")
    
    select {
    case m.triggerCh <- struct{}{}:
        // 触发信号已发送
    default:
        // 通道已满，说明已有触发信号
    }
}

// IsRebalancing 返回是否正在再平衡
func (m *Manager) IsRebalancing() bool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.isRebalancing
}

// GetStatus 获取负载均衡状态
func (m *Manager) GetStatus() map[string]interface{} {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    activeTasks := m.migrator.GetAllActiveTasks()
    
    return map[string]interface{}{
        "is_rebalancing":     m.isRebalancing,
        "last_rebalance":     m.lastRebalance,
        "active_tasks_count": len(activeTasks),
        "active_tasks":       activeTasks,
    }
}

// UpdateNodeMetrics 更新节点度量指标
func (m *Manager) UpdateNodeMetrics(nodeID string, metrics *types.NodeMetrics) {
    m.metricCollector.UpdateNodeMetrics(nodeID, metrics)
}

// GetNodeMetrics 获取指定节点的性能指标
func (m *Manager) GetNodeMetrics(nodeID string) *types.NodeMetrics {
    m.metricsLock.RLock()
    defer m.metricsLock.RUnlock()
    
    if metrics, exists := m.nodeMetrics[nodeID]; exists {
        // 返回指标副本以避免并发修改问题
        metricsCopy := *metrics
        return &metricsCopy
    }
    
    return nil
}

// 运行评估循环
func (m *Manager) runEvaluationLoop() {
    ticker := time.NewTicker(m.cfg.EvaluationInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-m.ctx.Done():
            return
        case <-ticker.C:
            // 周期性评估
            m.evaluateAndRebalance()
        case <-m.triggerCh:
            // 手动触发评估
            m.evaluateAndRebalance()
        }
    }
}

// 评估并执行再平衡
func (m *Manager) evaluateAndRebalance() {
    m.mu.Lock()
    
    // 如果已经在进行再平衡，则跳过
    if m.isRebalancing {
        m.mu.Unlock()
        m.logger.Info("已有再平衡任务在执行，跳过本次评估")
        return
    }
    
    // 检查距离上次再平衡的时间间隔
    if !m.lastRebalance.IsZero() && time.Since(m.lastRebalance) < m.cfg.MinMigrationInterval {
        m.mu.Unlock()
        m.logger.Info("距离上次再平衡时间不足，跳过本次评估",
            "last", m.lastRebalance,
            "min_interval", m.cfg.MinMigrationInterval)
        return
    }
    
    // 设置再平衡状态
    m.isRebalancing = true
    m.mu.Unlock()
    
    // 在函数退出时重置状态
    defer func() {
        m.mu.Lock()
        m.isRebalancing = false
        m.mu.Unlock()
    }()
    
    // 获取所有节点指标
    nodeMetrics := m.metricCollector.GetAllMetrics()
    if len(nodeMetrics) < 2 {
        m.logger.Info("节点数量不足，无需再平衡", "node_count", len(nodeMetrics))
        return
    }
    
    // 评估是否需要再平衡
    needRebalance, imbalanceScore := m.strategy.Evaluate(nodeMetrics)
    m.logger.Info("负载均衡评估结果",
        "need_rebalance", needRebalance,
        "imbalance_score", imbalanceScore,
        "threshold", m.cfg.ImbalanceThreshold)
    
    if !needRebalance {
        return
    }
    
    // 执行再平衡
    err := m.performRebalance(nodeMetrics)
    if err != nil {
        m.logger.Error("执行负载均衡失败", "error", err)
        return
    }
    
    // 更新最后再平衡时间
    m.mu.Lock()
    m.lastRebalance = time.Now()
    m.mu.Unlock()
    
    m.logger.Info("负载均衡计划已提交")
}

// 执行再平衡
func (m *Manager) performRebalance(nodeMetrics map[string]*types.NodeMetrics) error {
    // 生成迁移计划
    plans, err := m.strategy.GeneratePlan(nodeMetrics)
    if err != nil {
        return err
    }
    
    if len(plans) == 0 {
        m.logger.Info("没有需要执行的迁移计划")
        return nil
    }
    
    m.logger.Info("生成迁移计划", "plan_count", len(plans))
    
    // 提交迁移任务
    taskIDs := m.migrator.SubmitTasks(plans)
    m.logger.Info("已提交迁移任务", "task_count", len(taskIDs))
    
    return nil
}