package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/22827099/DFS_v1/common/logging"
	"github.com/22827099/DFS_v1/internal/metaserver/core/database"
)

// TransactionState 表示事务状态
type TransactionState string

const (
	TxActive    TransactionState = "active"
	TxCommitted TransactionState = "committed"
	TxAborted   TransactionState = "aborted"
)

// Transaction 表示一个元数据事务
type Transaction struct {
	ID        string
	Tx        *sql.Tx
	State     TransactionState
	StartTime time.Time
	Timeout   time.Duration
}

// Manager 事务管理器
type Manager struct {
	db        *database.Manager
	logger    logging.Logger
	txMap     sync.Map
	cleanupCh chan struct{}
	txTimeout time.Duration
}

// NewManager 创建新的事务管理器
func NewManager(db *database.Manager, logger logging.Logger) (*Manager, error) {
	if db == nil {
		return nil, errors.New("数据库管理器不能为空")
	}

	return &Manager{
		db:        db,
		logger:    logger,
		cleanupCh: make(chan struct{}),
		txTimeout: 5 * time.Minute, // 事务的最长运行时间
	}, nil
}

// Start 启动事务管理器
func (m *Manager) Start() error {
	m.logger.Info("启动事务管理器")
	go m.cleanupExpiredTransactions()
	return nil
}

// Stop 停止事务管理器
func (m *Manager) Stop(ctx context.Context) error {
	m.logger.Info("停止事务管理器")
	close(m.cleanupCh)

	// 回滚所有活跃事务
	m.txMap.Range(func(key, value interface{}) bool {
		txID := key.(string)
		tx := value.(*Transaction)

		if tx.State == TxActive {
			m.logger.Warn("关闭时回滚未完成事务: %s", txID)
			_ = tx.Tx.Rollback()
		}
		return true
	})

	return nil
}

// 清理过期事务
func (m *Manager) cleanupExpiredTransactions() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			m.txMap.Range(func(key, value interface{}) bool {
				txID := key.(string)
				tx := value.(*Transaction)

				// 只处理活跃事务
				if tx.State != TxActive {
					return true
				}

				// 检查是否超时
				if now.Sub(tx.StartTime) > tx.Timeout {
					m.logger.Warn("事务超时，自动回滚: %s", txID)
					_ = tx.Tx.Rollback()
					tx.State = TxAborted
				}

				return true
			})
		case <-m.cleanupCh:
			return
		}
	}
}

// Begin 开始新事务
func (m *Manager) Begin(ctx context.Context) (string, error) {
	tx, err := m.db.GetTx(ctx)
	if err != nil {
		return "", fmt.Errorf("开始事务失败: %w", err)
	}

	txID := fmt.Sprintf("tx-%d", time.Now().UnixNano())
	transaction := &Transaction{
		ID:        txID,
		Tx:        tx,
		State:     TxActive,
		StartTime: time.Now(),
		Timeout:   m.txTimeout,
	}

	m.txMap.Store(txID, transaction)
	m.logger.Debug("开始事务: %s", txID)

	return txID, nil
}

// Commit 提交事务
func (m *Manager) Commit(ctx context.Context, txID string) error {
	value, ok := m.txMap.Load(txID)
	if !ok {
		return fmt.Errorf("事务不存在: %s", txID)
	}

	tx := value.(*Transaction)
	if tx.State != TxActive {
		return fmt.Errorf("事务状态无效: %s, 状态=%s", txID, tx.State)
	}

	if err := tx.Tx.Commit(); err != nil {
		tx.State = TxAborted
		m.logger.Error("提交事务失败: %s, 错误=%v", txID, err)
		return fmt.Errorf("提交事务失败: %w", err)
	}

	tx.State = TxCommitted
	m.logger.Debug("提交事务: %s", txID)

	// 清理已完成的事务
	m.txMap.Delete(txID)
	return nil
}

// Rollback 回滚事务
func (m *Manager) Rollback(ctx context.Context, txID string) error {
	value, ok := m.txMap.Load(txID)
	if !ok {
		return fmt.Errorf("事务不存在: %s", txID)
	}

	tx := value.(*Transaction)
	if tx.State != TxActive {
		return fmt.Errorf("事务状态无效: %s, 状态=%s", txID, tx.State)
	}

	if err := tx.Tx.Rollback(); err != nil {
		m.logger.Error("回滚事务失败: %s, 错误=%v", txID, err)
		return fmt.Errorf("回滚事务失败: %w", err)
	}

	tx.State = TxAborted
	m.logger.Debug("回滚事务: %s", txID)

	// 清理已完成的事务
	m.txMap.Delete(txID)
	return nil
}

// WithTransaction 在事务中执行操作
func (m *Manager) WithTransaction(ctx context.Context, fn func(ctx context.Context, tx *sql.Tx) error) error {
	txID, err := m.Begin(ctx)
	if err != nil {
		return err
	}

	value, _ := m.txMap.Load(txID)
	tx := value.(*Transaction)

	// 确保事务最终会被处理
	defer func() {
		if tx.State == TxActive {
			_ = m.Rollback(ctx, txID)
		}
	}()

	// 执行事务回调
	err = fn(ctx, tx.Tx)
	if err != nil {
		m.logger.Warn("事务操作失败，回滚: %s, 错误=%v", txID, err)
		if rbErr := m.Rollback(ctx, txID); rbErr != nil {
			m.logger.Error("回滚事务失败: %s, 错误=%v", txID, rbErr)
		}
		return err
	}

	// 提交事务
	if err := m.Commit(ctx, txID); err != nil {
		m.logger.Error("提交事务失败: %s, 错误=%v", txID, err)
		return err
	}

	return nil
}
