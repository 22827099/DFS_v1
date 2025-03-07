package server

import (
	"context"
	"fmt"

	"github.com/22827099/DFS_v1/common/logging"
	httplib "github.com/22827099/DFS_v1/common/network/http"
	"github.com/22827099/DFS_v1/internal/metaserver/config"
	"github.com/22827099/DFS_v1/internal/metaserver/core"
)

// MetadataServer 表示元数据服务器实例
type MetadataServer struct {
	config     *config.Config
	logger     logging.Logger
	httpServer *httplib.Server
	core       *core.MetaCore
}

// NewServer 创建新的元数据服务器实例
func NewServer(cfg *config.Config, logger logging.Logger) (*MetadataServer, error) {
	// 初始化核心组件
	metaCore, err := core.NewMetaCore(cfg, logger)
	if err != nil {
		return nil, err
	}

	// 创建HTTP服务器
	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)
	httpServer := httplib.NewServer(serverAddr,
		httplib.WithServerLogger(logger),
		httplib.WithReadTimeout(cfg.Server.ReadTimeout),
		httplib.WithWriteTimeout(cfg.Server.WriteTimeout),
	)

	server := &MetadataServer{
		config:     cfg,
		logger:     logger,
		httpServer: httpServer,
		core:       metaCore,
	}

	// 注册API路由
	server.registerRoutes()

	return server, nil
}

// Start 启动服务器
func (s *MetadataServer) Start() error {
	s.logger.Info("元数据服务器启动于 %s:%d", s.config.Server.Address, s.config.Server.Port)

	// 启动核心服务
	if err := s.core.Start(); err != nil {
		return err
	}

	// 启动HTTP服务器
	if s.config.Security.EnableTLS {
		return s.httpServer.StartTLS(s.config.Security.CertFile, s.config.Security.KeyFile)
	}
	return s.httpServer.Start()
}

// Stop 停止服务器
func (s *MetadataServer) Stop(ctx context.Context) error {
	// 停止HTTP服务器
	if err := s.httpServer.Stop(ctx); err != nil {
		s.logger.Error("停止HTTP服务器错误: %v", err)
	}

	// 停止核心服务
	return s.core.Stop(ctx)
}

// registerRoutes 注册所有API路由
func (s *MetadataServer) registerRoutes() {
	// API根路径组
	api := s.httpServer.Router().Group("/api/v1")

	// 文件和目录操作
	api.GET("/metadata/{path}", s.handleGetMetadata)
	api.PUT("/metadata/{path}", s.handleCreateMetadata)
	api.DELETE("/metadata/{path}", s.handleDeleteMetadata)

	// 集群管理API
	cluster := api.Group("/cluster")
	cluster.GET("/status", s.handleClusterStatus)

	// 系统状态API
	s.httpServer.GET("/status", s.handleSystemStatus)
}

// API处理函数
func (s *MetadataServer) handleGetMetadata(c *httplib.Context) {
	// 将在处理器文件中实现
}

func (s *MetadataServer) handleCreateMetadata(c *httplib.Context) {
	// 将在处理器文件中实现
}

func (s *MetadataServer) handleDeleteMetadata(c *httplib.Context) {
	// 将在处理器文件中实现
}

func (s *MetadataServer) handleClusterStatus(c *httplib.Context) {
	// 将在处理器文件中实现
}

func (s *MetadataServer) handleSystemStatus(c *httplib.Context) {
	httplib.WriteSuccess(c.Response, map[string]string{
		"status":  "running",
		"version": "1.0.0",
	})
}
