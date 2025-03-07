package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	// 导入数据库驱动
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/22827099/DFS_v1/common/logging"
	"github.com/22827099/DFS_v1/internal/metaserver/config"
)

// Manager 管理数据库连接和操作
type Manager struct {
	config config.DatabaseConfig
	logger logging.Logger
	db     *sql.DB
	schema *Schema
}

// NewManager 创建新的数据库管理器
func NewManager(config config.DatabaseConfig, logger logging.Logger) (*Manager, error) {
	manager := &Manager{
		config: config,
		logger: logger,
	}

	return manager, nil
}

// Start 启动数据库管理器
func (m *Manager) Start() error {
	m.logger.Info("正在初始化数据库连接...")

	// 构建数据库连接字符串
	var dataSourceName string
	var driverName string

	switch m.config.Type {
	case "mysql":
		driverName = "mysql"
		dataSourceName = fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local",
			m.config.User, m.config.Password, m.config.Host, m.config.Port, m.config.Database,
		)
	case "postgres", "postgresql":
		driverName = "postgres"
		dataSourceName = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			m.config.Host, m.config.Port, m.config.User, m.config.Password, m.config.Database,
		)
	case "sqlite", "sqlite3":
		driverName = "sqlite3"
		dataSourceName = m.config.Database
		if m.config.Database == ":memory:" {
			m.logger.Info("使用内存数据库")
		}
	default:
		return fmt.Errorf("不支持的数据库类型: %s", m.config.Type)
	}

	// 打开数据库连接
	var err error
	m.db, err = sql.Open(driverName, dataSourceName)
	if err != nil {
		return fmt.Errorf("无法连接到数据库: %w", err)
	}

	// 设置连接池
	m.db.SetMaxOpenConns(m.config.MaxOpenConns)
	m.db.SetMaxIdleConns(m.config.MaxIdleConns)
	m.db.SetConnMaxLifetime(time.Duration(m.config.ConnMaxLifetime) * time.Second)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := m.db.PingContext(ctx); err != nil {
		m.db.Close()
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	m.logger.Info("数据库连接已建立")

	// 初始化数据库模式
	m.schema = NewSchema(m.db, m.logger)
	if err := m.schema.Initialize(ctx); err != nil {
		m.db.Close()
		return fmt.Errorf("初始化数据库模式失败: %w", err)
	}

	return nil
}

// Stop 停止数据库管理器
func (m *Manager) Stop(ctx context.Context) error {
	if m.db != nil {
		m.logger.Info("正在关闭数据库连接...")
		if err := m.db.Close(); err != nil {
			return fmt.Errorf("关闭数据库连接时出错: %w", err)
		}
		m.logger.Info("数据库连接已关闭")
	}
	return nil
}

// DB 返回数据库连接
func (m *Manager) DB() *sql.DB {
	return m.db
}

// GetTx 开始新事务
func (m *Manager) GetTx(ctx context.Context) (*sql.Tx, error) {
	if m.db == nil {
		return nil, errors.New("数据库连接未初始化")
	}

	tx, err := m.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	})
	if err != nil {
		return nil, fmt.Errorf("无法开始事务: %w", err)
	}

	return tx, nil
}

// ExecContext 执行SQL语句，不返回结果
func (m *Manager) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if m.db == nil {
		return nil, errors.New("数据库连接未初始化")
	}

	m.logger.Debug("执行SQL: %s, 参数: %v", query, args)
	result, err := m.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("执行SQL失败: %w", err)
	}

	return result, nil
}

// QueryContext 执行查询并返回行
func (m *Manager) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if m.db == nil {
		return nil, errors.New("数据库连接未初始化")
	}

	m.logger.Debug("执行查询: %s, 参数: %v", query, args)
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("执行查询失败: %w", err)
	}

	return rows, nil
}

// QueryRowContext 执行查询并返回单行
func (m *Manager) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if m.db == nil {
		panic("数据库连接未初始化")
	}

	m.logger.Debug("执行单行查询: %s, 参数: %v", query, args)
	return m.db.QueryRowContext(ctx, query, args...)
}

// WithTransaction 在事务中执行函数
func (m *Manager) WithTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := m.GetTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			// 发生panic，回滚事务
			tx.Rollback()
			panic(r) // 重新抛出panic
		}
	}()

	if err := fn(tx); err != nil {
		// 发生错误，回滚事务
		if rbErr := tx.Rollback(); rbErr != nil {
			m.logger.Error("回滚事务失败: %v", rbErr)
		}
		return err
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}
