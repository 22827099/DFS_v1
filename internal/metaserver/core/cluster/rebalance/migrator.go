package rebalance

import (
	"context"
	"sync"
	"time"

	"github.com/22827099/DFS_v1/common/logging"
	"github.com/google/uuid"
)

// TaskState 表示迁移任务的状态
type TaskState string

const (
	TaskStatePending   TaskState = "pending"   // 等待执行
	TaskStateRunning   TaskState = "running"   // 正在执行
	TaskStateCompleted TaskState = "completed" // 已完成
	TaskStateFailed    TaskState = "failed"    // 失败
)

// MigrationTask 数据迁移任务
type MigrationTask struct {
	TaskID      string         `json:"task_id"`      // 任务ID
	Plan        *MigrationPlan `json:"plan"`         // 迁移计划
	State       TaskState      `json:"state"`        // 任务状态
	Progress    float64        `json:"progress"`     // 进度（0-100）
	StartTime   time.Time      `json:"start_time"`   // 开始时间
	EndTime     time.Time      `json:"end_time"`     // 结束时间
	ErrorDetail string         `json:"error_detail"` // 错误详情
}

// Migrator 数据迁移器
type Migrator struct {
	ctx           context.Context     // 上下文，用于控制整个迁移器生命周期
	maxConcurrent int                 // 最大并发迁移任务数
	logger        logging.Logger      // 日志器
	tasks         sync.Map            // 所有任务映射，使用sync.Map减少锁竞争
	pendingTasks  chan *MigrationTask // 等待执行的任务队列
	wg            sync.WaitGroup      // 等待所有任务完成
}

// NewMigrator 创建新的数据迁移器
func NewMigrator(ctx context.Context, maxConcurrent int, logger logging.Logger) *Migrator {
	if maxConcurrent <= 0 {
		maxConcurrent = 5 // 默认最大并发数
	}

	return &Migrator{
		ctx:           ctx,
		maxConcurrent: maxConcurrent,
		logger:        logger.WithContext(map[string]interface{}{"component": "migrator"}),
		pendingTasks:  make(chan *MigrationTask, 100), // 缓冲区大小可调整
	}
}

// Start 启动迁移器
func (m *Migrator) Start() {
	m.logger.Info("启动数据迁移器", "max_concurrent", m.maxConcurrent)

	// 启动worker池
	for i := 0; i < m.maxConcurrent; i++ {
		m.wg.Add(1)
		go m.worker(i)
	}
}

// Stop 停止迁移器
func (m *Migrator) Stop() {
	m.logger.Info("停止数据迁移器")

	// 等待所有任务完成
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	// 设置超时，防止长时间阻塞
	select {
	case <-done:
		m.logger.Info("所有迁移任务已完成")
	case <-time.After(30 * time.Second):
		m.logger.Warn("等待迁移任务完成超时")
	}
}

// SubmitTasks 提交迁移任务
func (m *Migrator) SubmitTasks(plans []*MigrationPlan) []string {
	taskIDs := make([]string, 0, len(plans))

	for _, plan := range plans {
		taskID := uuid.New().String()

		task := &MigrationTask{
			TaskID:    taskID,
			Plan:      plan,
			State:     TaskStatePending,
			Progress:  0,
			StartTime: time.Time{},
			EndTime:   time.Time{},
		}

		m.tasks.Store(taskID, task)

		// 非阻塞地发送到任务队列
		select {
		case m.pendingTasks <- task:
			// 成功添加到队列
		default:
			// 队列已满，改变任务状态为失败
			task.State = TaskStateFailed
			task.ErrorDetail = "任务队列已满"
			m.logger.Warn("任务队列已满，无法提交新任务", "task_id", taskID)
		}

		taskIDs = append(taskIDs, taskID)

		m.logger.Info("提交迁移任务",
			"task_id", taskID,
			"source", plan.SourceNodeID,
			"target", plan.TargetNodeID,
			"shards", len(plan.ShardIDs),
			"bytes", plan.EstimatedBytes)
	}

	return taskIDs
}

// GetTaskStatus 获取任务状态
func (m *Migrator) GetTaskStatus(taskID string) (*MigrationTask, bool) {
	if value, exists := m.tasks.Load(taskID); exists {
		task := value.(*MigrationTask)
		// 返回副本以避免并发修改
		taskCopy := *task
		return &taskCopy, true
	}
	return nil, false
}

// GetAllActiveTasks 获取所有活动任务
func (m *Migrator) GetAllActiveTasks() []*MigrationTask {
	activeTasks := make([]*MigrationTask, 0)

	m.tasks.Range(func(key, value interface{}) bool {
		task := value.(*MigrationTask)
		if task.State == TaskStatePending || task.State == TaskStateRunning {
			// 返回副本以避免并发修改
			taskCopy := *task
			activeTasks = append(activeTasks, &taskCopy)
		}
		return true
	})

	return activeTasks
}

// worker 工作协程，处理迁移任务
func (m *Migrator) worker(id int) {
	defer m.wg.Done()

	m.logger.Info("启动迁移工作协程", "worker_id", id)

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Info("迁移工作协程退出", "worker_id", id)
			return
		case task := <-m.pendingTasks:
			// 处理任务
			m.processTask(task)
		}
	}
}

// processTask 处理迁移任务
func (m *Migrator) processTask(task *MigrationTask) {
	// 更新任务状态为运行中
	task.State = TaskStateRunning
	task.StartTime = time.Now()
	m.tasks.Store(task.TaskID, task)

	m.logger.Info("开始处理迁移任务",
		"task_id", task.TaskID,
		"source", task.Plan.SourceNodeID,
		"target", task.Plan.TargetNodeID)

	// 模拟迁移过程
	success := m.executeMigration(task)

	// 完成时间
	task.EndTime = time.Now()

	if success {
		task.State = TaskStateCompleted
		task.Progress = 100
		m.logger.Info("迁移任务完成",
			"task_id", task.TaskID,
			"duration", task.EndTime.Sub(task.StartTime))
	} else {
		task.State = TaskStateFailed
		if task.ErrorDetail == "" {
			task.ErrorDetail = "迁移过程中断"
		}
		m.logger.Error("迁移任务失败",
			"task_id", task.TaskID,
			"error", task.ErrorDetail)
	}

	// 更新任务状态
	m.tasks.Store(task.TaskID, task)
}

// executeMigration 执行迁移操作
func (m *Migrator) executeMigration(task *MigrationTask) bool {
	// 这里应该实现实际的迁移逻辑
	// 当前是模拟实现，实际项目中需要对接存储层API

	// 模拟迁移进度
	totalShards := len(task.Plan.ShardIDs)
	if totalShards == 0 {
		task.ErrorDetail = "没有要迁移的分片"
		return false
	}

	// 为每个分片分配时间
	timePerShard := 2 * time.Second

	for i, shardID := range task.Plan.ShardIDs {
		// 检查是否被取消
		select {
		case <-m.ctx.Done():
			task.ErrorDetail = "迁移任务被取消"
			return false
		default:
			// 继续执行
		}

		// 模拟分片迁移
		m.logger.Debug("迁移分片",
			"task_id", task.TaskID,
			"shard_id", shardID,
			"progress", float64(i)/float64(totalShards)*100)

		// 更新进度
		task.Progress = float64(i) / float64(totalShards) * 100
		m.tasks.Store(task.TaskID, task)

		// 模拟迁移时间
		select {
		case <-time.After(timePerShard):
			// 分片迁移完成
		case <-m.ctx.Done():
			task.ErrorDetail = "迁移任务被取消"
			return false
		}
	}

	// 迁移完成
	return true
}

// CancelTask 取消任务
func (m *Migrator) CancelTask(taskID string) bool {
	value, exists := m.tasks.Load(taskID)
	if !exists {
		return false
	}

	task := value.(*MigrationTask)
	if task.State != TaskStatePending && task.State != TaskStateRunning {
		return false // 只能取消等待或运行中的任务
	}

	task.State = TaskStateFailed
	task.ErrorDetail = "任务被手动取消"
	task.EndTime = time.Now()
	m.tasks.Store(taskID, task)

	m.logger.Info("取消迁移任务", "task_id", taskID)

	return true
}
