package database

import (
	"context"
	"fmt"
	"time"
)

// Migration 表示一个数据库迁移
type Migration struct {
	Version     int
	Description string
	SQL         string
}

// MigrationManager 管理数据库迁移
type MigrationManager struct {
	manager *Manager
}

// NewMigrationManager 创建新的迁移管理器
func NewMigrationManager(manager *Manager) *MigrationManager {
	return &MigrationManager{
		manager: manager,
	}
}

// ensureMigrationTable 确保迁移表存在
func (m *MigrationManager) ensureMigrationTable(ctx context.Context) error {
	_, err := m.manager.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version         INT PRIMARY KEY,
            description     VARCHAR(255) NOT NULL,
            applied_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        )
    `)
	return err
}

// GetAppliedMigrations 获取已应用的迁移列表
func (m *MigrationManager) GetAppliedMigrations(ctx context.Context) (map[int]time.Time, error) {
	if err := m.ensureMigrationTable(ctx); err != nil {
		return nil, err
	}

	rows, err := m.manager.QueryContext(ctx, `
        SELECT version, applied_at FROM schema_migrations ORDER BY version ASC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]time.Time)
	for rows.Next() {
		var version int
		var appliedAt time.Time
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return nil, err
		}
		applied[version] = appliedAt
	}

	return applied, rows.Err()
}

// ApplyMigrations 应用迁移
func (m *MigrationManager) ApplyMigrations(ctx context.Context, migrations []Migration) error {
	// 获取已应用的迁移
	applied, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	// 应用新迁移
	for _, migration := range migrations {
		if _, ok := applied[migration.Version]; ok {
			// 迁移已应用，跳过
			m.manager.logger.Info("迁移 %d (%s) 已应用，跳过", migration.Version, migration.Description)
			continue
		}

		m.manager.logger.Info("应用迁移 %d: %s", migration.Version, migration.Description)

		// 在事务中执行迁移
		err := m.manager.DoInTransaction(ctx, func(tx *Transaction) error {
			// 执行迁移SQL
			_, err := tx.Exec(ctx, migration.SQL)
			if err != nil {
				return err
			}

			// 记录迁移已应用
			_, err = tx.Exec(ctx, `
                INSERT INTO schema_migrations (version, description)
                VALUES (?, ?)
            `, migration.Version, migration.Description)

			return err
		})

		if err != nil {
			return fmt.Errorf("应用迁移 %d 失败: %w", migration.Version, err)
		}

		m.manager.logger.Info("迁移 %d 应用成功", migration.Version)
	}

	return nil
}
