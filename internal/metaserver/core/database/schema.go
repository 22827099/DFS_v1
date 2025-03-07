package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/22827099/DFS_v1/common/logging"
)

// Schema 定义数据库模式
type Schema struct {
	db     *sql.DB
	logger logging.Logger
}

// NewSchema 创建数据库模式管理器
func NewSchema(db *sql.DB, logger logging.Logger) *Schema {
	return &Schema{
		db:     db,
		logger: logger,
	}
}

// Initialize 初始化数据库模式
func (s *Schema) Initialize(ctx context.Context) error {
	s.logger.Info("初始化数据库模式...")

	// 创建迁移表
    migrationManager := NewMigrationManager(&Manager{db: s.db, logger: s.logger})
    if err := migrationManager.ensureMigrationTable(ctx); err != nil {
        return fmt.Errorf("创建迁移表失败: %w", err)
    }

	// 创建表
	for _, statement := range createTableStatements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("创建表失败: %w", err)
		}
	}

	// 创建索引
	for _, statement := range createIndexStatements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("创建索引失败: %w", err)
		}
	}

	// 初始化根目录
	if err := s.initRootDirectory(ctx); err != nil {
		return fmt.Errorf("初始化根目录失败: %w", err)
	}

	// 初始化系统用户
    if err := s.initSystemUser(ctx); err != nil {
        return fmt.Errorf("初始化系统用户失败: %w", err)
    }

	s.logger.Info("数据库模式初始化完成")
	return nil
}

// 初始化根目录
func (s *Schema) initRootDirectory(ctx context.Context) error {
	// 检查根目录是否已存在
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM directories WHERE parent_id IS NULL").Scan(&count)
	if err != nil {
		return err
	}

	// 根目录不存在，创建它
	if count == 0 {
		_, err := s.db.ExecContext(ctx, `
            INSERT INTO directories (dir_id, parent_id, name, owner_id)
            VALUES (1, NULL, '/', 1)
        `)
		return err
	}

	return nil
}

// 初始化系统用户
func (s *Schema) initSystemUser(ctx context.Context) error {
    // 检查系统用户是否已存在
    var count int
    err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE user_id = 1").Scan(&count)
    if err != nil {
        return err
    }

    // 系统用户不存在，创建它
    if count == 0 {
        // 使用预设的安全密码哈希和盐值
        _, err := s.db.ExecContext(ctx, `
            INSERT INTO users (user_id, username, password_hash, salt, created_at, status)
            VALUES (1, 'system', 'preset-secure-hash', 'preset-salt', ?, 'active')
        `, time.Now())
        return err
    }

    return nil
}

// 获取元数据节点结构版本
func (s *Schema) GetSchemaVersion(ctx context.Context) (int, error) {
    // 查询迁移表获取最高版本
    var version int
    err := s.db.QueryRowContext(ctx, `
        SELECT COALESCE(MAX(version), 0) FROM schema_migrations
    `).Scan(&version)
    
    return version, err
}

// 创建表的SQL语句
var createTableStatements = []string{
	// 目录表
	`CREATE TABLE IF NOT EXISTS directories (
        dir_id          BIGINT PRIMARY KEY,
        parent_id       BIGINT,
        name            VARCHAR(255) NOT NULL,
        owner_id        INT NOT NULL,
        created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        modified_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        is_deleted      BOOLEAN DEFAULT FALSE,
        mode            INT DEFAULT 755,
        UNIQUE (parent_id, name),
        FOREIGN KEY (parent_id) REFERENCES directories(dir_id)
    )`,

	// 文件表
	`CREATE TABLE IF NOT EXISTS files (
        file_id         BIGINT PRIMARY KEY,
        parent_dir_id   BIGINT NOT NULL,
        name            VARCHAR(255) NOT NULL,
        size            BIGINT DEFAULT 0,
        owner_id        INT NOT NULL,
        created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        modified_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        accessed_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        version         INT DEFAULT 1,
        is_deleted      BOOLEAN DEFAULT FALSE,
        checksum        VARCHAR(64),
        mode            INT DEFAULT 644,
        UNIQUE (parent_dir_id, name),
        FOREIGN KEY (parent_dir_id) REFERENCES directories(dir_id)
    )`,

	// 数据块表
	`CREATE TABLE IF NOT EXISTS chunks (
        chunk_id        BIGINT PRIMARY KEY,
        file_id         BIGINT NOT NULL,
        chunk_index     INT NOT NULL,
        size            INT NOT NULL DEFAULT 0,
        checksum        VARCHAR(64),
        UNIQUE (file_id, chunk_index),
        FOREIGN KEY (file_id) REFERENCES files(file_id)
    )`,

	// 数据节点表 (datanodes)
	`	CREATE TABLE datanodes (
		node_id         VARCHAR(64) PRIMARY KEY,
		address         VARCHAR(128) NOT NULL,
		port            INT NOT NULL,
		status          VARCHAR(16) NOT NULL DEFAULT 'active',
		capacity_total  BIGINT NOT NULL,
		capacity_used   BIGINT NOT NULL DEFAULT 0,
		last_heartbeat  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		rack_id         VARCHAR(64)
	)`,

	// 数据块副本表
	`	CREATE TABLE replicas (
		replica_id      BIGINT PRIMARY KEY,
		chunk_id        BIGINT NOT NULL,
		node_id         VARCHAR(64) NOT NULL,
		status          VARCHAR(16) NOT NULL DEFAULT 'valid',
		created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		
		FOREIGN KEY (chunk_id) REFERENCES chunks(chunk_id),
		FOREIGN KEY (node_id) REFERENCES datanodes(node_id)
	)`,

	// 用户表（users）
	`	CREATE TABLE users (
		user_id         INT PRIMARY KEY,
		username        VARCHAR(64) NOT NULL UNIQUE,
		password_hash   VARCHAR(128) NOT NULL,
		salt            VARCHAR(32) NOT NULL,
		created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		status          VARCHAR(16) NOT NULL DEFAULT 'active'
	)`,
	
	//
	`   CREATE TABLE permissions (
		permission_id   BIGINT PRIMARY KEY,
		object_id       BIGINT NOT NULL,
		object_type     VARCHAR(16) NOT NULL,  -- 'file' or 'directory'
		user_id         INT NOT NULL,
		permission_type VARCHAR(16) NOT NULL,  -- 'read', 'write', 'execute'
		
		UNIQUE (object_id, object_type, user_id, permission_type),
		FOREIGN KEY (user_id) REFERENCES users(user_id)
	)`,
}

// 创建索引的SQL语句
var createIndexStatements = []string{
	`CREATE INDEX IF NOT EXISTS idx_dirs_parent ON directories(parent_id)`,
	`CREATE INDEX IF NOT EXISTS idx_files_parent ON files(parent_dir_id)`,
	`CREATE INDEX IF NOT EXISTS idx_chunks_file ON chunks(file_id)`,
	`CREATE INDEX IF NOT EXISTS idx_replicas_chunk ON replicas(chunk_id)`,
	`CREATE INDEX IF NOT EXISTS idx_replicas_node ON replicas(node_id)`,
	`CREATE INDEX idx_users_username ON users(username)`,
	`CREATE INDEX idx_permissions_object ON permissions(object_id, object_type)`,
	`CREATE INDEX idx_permissions_user ON permissions(user_id)`,

	// 其他索引的创建语句...
}
