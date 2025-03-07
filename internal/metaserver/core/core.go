package core

import (
	"context"

	"github.com/22827099/DFS_v1/common/logging"
	"github.com/22827099/DFS_v1/internal/metaserver/config"
	// "github.com/22827099/DFS_v1/internal/metaserver/core/cluster"
	"github.com/22827099/DFS_v1/internal/metaserver/core/database"
	"github.com/22827099/DFS_v1/internal/metaserver/core/metadata"
)

// MetaCore 封装元数据服务器的核心功能
type MetaCore struct {
	config  *config.Config
	logger  logging.Logger
	db      *database.Manager
	meta    *metadata.Manager
	cluster *cluster.Manager
}

// NewMetaCore 创建核心组件管理器
func NewMetaCore(cfg *config.Config, logger logging.Logger) (*MetaCore, error) {
	// 初始化数据库
	db, err := database.NewManager(cfg.Database, logger)
	if err != nil {
		return nil, err
	}

	// 初始化元数据管理
	meta, err := metadata.NewManager(db, logger)
	if err != nil {
		return nil, err
	}

	// 初始化集群管理
	clusterMgr, err := cluster.NewManager(cfg.Cluster, logger)
	if err != nil {
		return nil, err
	}

	return &MetaCore{
		config:  cfg,
		logger:  logger,
		db:      db,
		meta:    meta,
		cluster: clusterMgr,
	}, nil
}

// Start 启动所有核心组件
func (c *MetaCore) Start() error {
	// 启动数据库连接
	if err := c.db.Start(); err != nil {
		return err
	}

	// 启动元数据管理
	if err := c.meta.Start(); err != nil {
		return err
	}

	// 启动集群管理
	return c.cluster.Start()
}

// Stop 停止所有核心组件
func (c *MetaCore) Stop(ctx context.Context) error {
	// 停止集群管理
	if err := c.cluster.Stop(ctx); err != nil {
		c.logger.Error("停止集群管理错误: %v", err)
	}

	// 停止元数据管理
	if err := c.meta.Stop(ctx); err != nil {
		c.logger.Error("停止元数据管理错误: %v", err)
	}

	// 停止数据库连接
	return c.db.Stop(ctx)
}
