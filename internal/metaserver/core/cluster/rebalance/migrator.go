package rebalance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/22827099/DFS_v1/common/logging"
	"github.com/google/uuid"
)

// TaskStatus 表示迁移任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 等待执行
	TaskStatusRunning   TaskStatus = "running"   // 正在执行
	TaskStatusCompleted TaskStatus = "completed" // 已完成
	TaskStatusFailed    TaskStatus = "failed"    // 失败
	TaskStatusCancelled TaskStatus = "cancelled" // 已取消
)

// MigrationTask 表示数据迁移任务
type MigrationTask struct {
	TaskID        string         `json:"task_id"`        // 任务ID
	Plan          *MigrationPlan `json:"plan"`           // 关联的迁移计划
	Status        TaskStatus     `json:"status"`         // 任务状态
	StartTime     time.Time      `json:"start_time"`     // 开始时间
	EndTime       time.Time      `json:"end_time"`       // 结束时间
	Progress      float64        `json:"progress"`       // 进度(0-100)
	FailureReason string         `json:"failure_reason"` // 失败原因
	RetryCount    int            `json:"retry_count"`    // 重试次数
	BytesMoved    uint64         `json:"bytes_moved"`    // 已迁移数据量
}

// TaskResult 表示任务执行结果
type TaskResult struct {
	TaskID        string
	Success       bool
	FailureReason string
	BytesMoved    uint64
}

// Migrator 数据迁移器
type Migrator struct {
	ctx                context.Context
	cancel             context.CancelFunc
	maxConcurrentTasks int
	logger             logging.Logger
	tasks              map[string]*MigrationTask
	taskQueue          []*MigrationTask
	activeTasks        map[string]struct{}
	mu                 sync.RWMutex
	workerWg           sync.WaitGroup
	taskResultCh       chan TaskResult
	dataNodeClient     DataNodeClient // 用于与数据节点通信
	isRunning          bool
	maxRetries         int
}

// DataNodeClient 数据节点客户端接口
type DataNodeClient interface {
	// TransferData 在两个数据节点间传输数据
	TransferData(ctx context.Context, sourceNodeID, targetNodeID string, shardIDs []string) (uint64, error)
	// ValidateTransferResult 验证传输结果
	ValidateTransferResult(ctx context.Context, sourceNodeID, targetNodeID string, shardIDs []string) error
}

// NewMigrator 创建新的数据迁移器
func NewMigrator(ctx context.Context, maxConcurrentTasks int, logger logging.Logger) *Migrator {
	migrationCtx, cancel := context.WithCancel(ctx)

	return &Migrator{
		ctx:                migrationCtx,
		cancel:             cancel,
		maxConcurrentTasks: maxConcurrentTasks,
		logger:             logger.WithContext(map[string]interface{}{"component": "migrator"}),
		tasks:              make(map[string]*MigrationTask),
		activeTasks:        make(map[string]struct{}),
		taskResultCh:       make(chan TaskResult, 100),
		maxRetries:         3,
		// 默认情况下使用一个模拟的数据节点客户端
		// 实际使用中应注入真实的客户端
		dataNodeClient: &mockDataNodeClient{},
	}
}

// SetDataNodeClient 设置数据节点客户端
func (m *Migrator) SetDataNodeClient(client DataNodeClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dataNodeClient = client
}

// Start 启动迁移器
func (m *Migrator) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return
	}

	m.logger.Info("启动数据迁移器")
	m.isRunning = true

	// 启动任务处理器
	go m.taskProcessor()
	// 启动结果处理器
	go m.resultProcessor()
}

// Stop 停止迁移器
func (m *Migrator) Stop() {
	m.mu.Lock()
	if !m.isRunning {
		m.mu.Unlock()
		return
	}

	m.isRunning = false
	m.cancel() // 取消所有操作的上下文
	m.mu.Unlock()

	// 等待所有工作协程退出
	m.workerWg.Wait()
	m.logger.Info("数据迁移器已停止")
}

// SubmitTasks 提交迁移任务，返回任务ID列表
func (m *Migrator) SubmitTasks(plans []*MigrationPlan) []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	taskIDs := make([]string, 0, len(plans))

	// 根据优先级排序计划
	sortPlans(plans)

	for _, plan := range plans {
		taskID := uuid.New().String()

		task := &MigrationTask{
			TaskID:     taskID,
			Plan:       plan,
			Status:     TaskStatusPending,
			Progress:   0.0,
			RetryCount: 0,
		}

		m.tasks[taskID] = task
		m.taskQueue = append(m.taskQueue, task)
		taskIDs = append(taskIDs, taskID)

		m.logger.Info("已提交迁移任务",
			"task_id", taskID,
			"source", plan.SourceNodeID,
			"target", plan.TargetNodeID)
	}

	return taskIDs
}

// CancelTask 取消指定任务
func (m *Migrator) CancelTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("任务不存在: %s", taskID)
	}

	if task.Status != TaskStatusPending && task.Status != TaskStatusRunning {
		return fmt.Errorf("无法取消状态为 %s 的任务", task.Status)
	}

	// 如果任务正在运行，标记为取消但让它完成当前操作
	// 如果任务在队列中，直接移除
	if task.Status == TaskStatusPending {
		// 从队列中移除
		for i, t := range m.taskQueue {
			if t.TaskID == taskID {
				m.taskQueue = append(m.taskQueue[:i], m.taskQueue[i+1:]...)
				break
			}
		}
	}

	task.Status = TaskStatusCancelled
	task.EndTime = time.Now()

	m.logger.Info("任务已取消", "task_id", taskID)
	return nil
}

// GetTask 获取任务详情
func (m *Migrator) GetTask(taskID string) (*MigrationTask, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("任务不存在: %s", taskID)
	}

	// 返回任务的副本避免并发修改
	taskCopy := *task
	return &taskCopy, nil
}

// GetAllTasks 获取所有任务
func (m *Migrator) GetAllTasks() []*MigrationTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]*MigrationTask, 0, len(m.tasks))
	for _, task := range m.tasks {
		taskCopy := *task
		tasks = append(tasks, &taskCopy)
	}

	return tasks
}

// GetAllActiveTasks 获取所有活跃任务
func (m *Migrator) GetAllActiveTasks() []*MigrationTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]*MigrationTask, 0, len(m.activeTasks))
	for taskID := range m.activeTasks {
		if task, exists := m.tasks[taskID]; exists {
			taskCopy := *task
			tasks = append(tasks, &taskCopy)
		}
	}

	return tasks
}

// 任务处理器 - 调度并执行迁移任务
func (m *Migrator) taskProcessor() {
	m.workerWg.Add(1)
	defer m.workerWg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.scheduleNextTasks()
		}
	}
}

// 结果处理器 - 处理任务完成结果
func (m *Migrator) resultProcessor() {
	m.workerWg.Add(1)
	defer m.workerWg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return
		case result := <-m.taskResultCh:
			m.handleTaskResult(result)
		}
	}
}

// 调度下一批任务
func (m *Migrator) scheduleNextTasks() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果没有运行，不调度任务
	if !m.isRunning {
		return
	}

	// 计算可启动的任务数量
	availableSlots := m.maxConcurrentTasks - len(m.activeTasks)
	if availableSlots <= 0 || len(m.taskQueue) == 0 {
		return
	}

	// 启动任务，但不超过可用槽位数量
	tasksToStart := min(availableSlots, len(m.taskQueue))
	for i := 0; i < tasksToStart; i++ {
		task := m.taskQueue[0]
		m.taskQueue = m.taskQueue[1:] // 从队列中移除

		task.Status = TaskStatusRunning
		task.StartTime = time.Now()
		m.activeTasks[task.TaskID] = struct{}{}

		// 启动一个协程执行任务
		go m.executeTask(task)
	}
}

// 执行任务
func (m *Migrator) executeTask(task *MigrationTask) {
	m.workerWg.Add(1)
	defer m.workerWg.Done()

	m.logger.Info("开始执行迁移任务",
		"task_id", task.TaskID,
		"source", task.Plan.SourceNodeID,
		"target", task.Plan.TargetNodeID)

	// 创建任务上下文，如果迁移器被停止，此上下文将被取消
	taskCtx, cancel := context.WithTimeout(m.ctx, 2*time.Hour)
	defer cancel()

	// 执行实际的数据传输
	bytesMoved, err := m.dataNodeClient.TransferData(
		taskCtx,
		task.Plan.SourceNodeID,
		task.Plan.TargetNodeID,
		task.Plan.ShardIDs,
	)

	result := TaskResult{
		TaskID:     task.TaskID,
		Success:    err == nil,
		BytesMoved: bytesMoved,
	}

	if err != nil {
		result.FailureReason = err.Error()
		m.logger.Error("迁移任务执行失败",
			"task_id", task.TaskID,
			"error", err)
	} else {
		// 验证传输结果
		err = m.dataNodeClient.ValidateTransferResult(
			taskCtx,
			task.Plan.SourceNodeID,
			task.Plan.TargetNodeID,
			task.Plan.ShardIDs,
		)

		if err != nil {
			result.Success = false
			result.FailureReason = fmt.Sprintf("验证传输结果失败: %v", err)
			m.logger.Error("验证迁移结果失败",
				"task_id", task.TaskID,
				"error", err)
		} else {
			m.logger.Info("迁移任务执行成功",
				"task_id", task.TaskID,
				"bytes_moved", bytesMoved)
		}
	}

	// 发送任务结果
	select {
	case m.taskResultCh <- result:
		// 结果发送成功
	case <-m.ctx.Done():
		// 如果上下文已取消，则忽略结果
	}
}

// 处理任务结果
func (m *Migrator) handleTaskResult(result TaskResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[result.TaskID]
	if !exists {
		m.logger.Warn("收到未知任务的结果", "task_id", result.TaskID)
		return
	}

	// 从活跃任务列表中移除
	delete(m.activeTasks, task.TaskID)

	// 更新任务状态
	task.BytesMoved = result.BytesMoved
	task.EndTime = time.Now()

	if result.Success {
		// 任务成功
		task.Status = TaskStatusCompleted
		task.Progress = 100.0
	} else {
		// 任务失败，检查是否需要重试
		task.FailureReason = result.FailureReason

		if task.RetryCount < m.maxRetries {
			// 重试任务
			task.RetryCount++
			task.Status = TaskStatusPending
			task.Progress = 0.0

			// 添加回队列，但放在后面
			m.taskQueue = append(m.taskQueue, task)

			m.logger.Info("任务已重新排队等待重试",
				"task_id", task.TaskID,
				"retry", task.RetryCount,
				"max_retries", m.maxRetries)
		} else {
			// 超过最大重试次数
			task.Status = TaskStatusFailed
			m.logger.Error("任务超过最大重试次数，标记为失败",
				"task_id", task.TaskID,
				"max_retries", m.maxRetries)
		}
	}
}

// 对迁移计划按优先级排序（优先级高的先执行）
func sortPlans(plans []*MigrationPlan) {
	// 使用冒泡排序，简单实现
	for i := 0; i < len(plans)-1; i++ {
		for j := 0; j < len(plans)-i-1; j++ {
			if plans[j].Priority < plans[j+1].Priority {
				plans[j], plans[j+1] = plans[j+1], plans[j]
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// mockDataNodeClient 是一个模拟的数据节点客户端，用于测试
type mockDataNodeClient struct{}

func (m *mockDataNodeClient) TransferData(ctx context.Context, sourceNodeID, targetNodeID string, shardIDs []string) (uint64, error) {
	// 模拟数据传输
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-time.After(5 * time.Second): // 模拟传输延迟
	}

	// 假设每个分片1GB
	bytesTransferred := uint64(len(shardIDs)) * 1024 * 1024 * 1024
	return bytesTransferred, nil
}

func (m *mockDataNodeClient) ValidateTransferResult(ctx context.Context, sourceNodeID, targetNodeID string, shardIDs []string) error {
	// 模拟验证过程
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1 * time.Second): // 模拟验证延迟
	}

	// 始终返回验证成功
	return nil
}
