package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/22827099/DFS_v1/common/config"
	"github.com/22827099/DFS_v1/common/errors"
	"github.com/22827099/DFS_v1/common/logging"
	nethttp "github.com/22827099/DFS_v1/common/network/http"
	"github.com/22827099/DFS_v1/internal/metaserver/core/cluster"
	"github.com/22827099/DFS_v1/internal/metaserver/core/metadata"
)

// MetadataServer 元数据服务器结构
type MetadataServer struct {
	config     *config.SystemConfig
	httpServer *nethttp.Server
	logger     logging.Logger
	metaStore  metadata.Store
	cluster    cluster.Manager
	mu         sync.RWMutex
	running    bool
}

// ServerOption 允许配置服务器的选项函数
type ServerOption func(*MetadataServer)

// NewServer 创建新的元数据服务器
func NewServer(cfg *config.SystemConfig, options ...ServerOption) (*MetadataServer, error) {
	if cfg == nil {
		return nil, errors.New(errors.InvalidArgument, "配置不能为空")
	}

	logger := logging.GetLogger("metaserver")
	server := &MetadataServer{
		config:  cfg,
		logger:  logger,
		running: false,
	}

	// 应用选项
	for _, option := range options {
		option(server)
	}

	// 如果没有提供元数据存储，创建默认的
	if server.metaStore == nil {
		metaStore, err := NewMemoryStore()
		if err != nil {
			return nil, errors.Wrap(err, errors.Internal, "初始化元数据存储失败")
		}
		server.metaStore = metaStore
	}

	// 如果没有提供集群管理器，创建默认的
	if server.cluster == nil {
		clusterMgr, err := cluster.NewManager(cfg.Cluster, logger)
		if err != nil {
			return nil, errors.Wrap(err, errors.Internal, "初始化集群管理器失败")
		}
		server.cluster = clusterMgr
	}

	// 创建HTTP服务器
	httpServer := nethttp.NewServer(fmt.Sprintf(":%d", 8080)) // 默认端口，实际应从配置中读取

	// 添加中间件
	httpServer.Use(nethttp.RequestIDMiddleware())
	httpServer.Use(nethttp.LoggingMiddleware(logger))
	httpServer.Use(nethttp.RecoveryMiddleware(logger))

	// 设置路由
	server.setupRoutes(httpServer)
	server.httpServer = httpServer

	return server, nil
}

// WithMetaStore 设置元数据存储
func WithMetaStore(store metadata.Store) ServerOption {
	return func(s *MetadataServer) {
		s.metaStore = store
	}
}

// WithClusterManager 设置集群管理器
func WithClusterManager(manager cluster.Manager) ServerOption {
	return func(s *MetadataServer) {
		s.cluster = manager
	}
}

// Start 启动服务器
func (s *MetadataServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return errors.New(errors.AlreadyExists, "服务器已经在运行")
	}

	// 初始化元数据存储
	if err := s.metaStore.Initialize(); err != nil {
		return errors.Wrap(err, errors.Internal, "初始化元数据存储失败")
	}

	// 启动集群服务
	if err := s.cluster.Start(); err != nil {
		return errors.Wrap(err, errors.Internal, "启动集群服务失败")
	}

	// 启动HTTP服务器
	go func() {
		if err := s.httpServer.Start(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP服务器异常退出: %v", err)
		}
	}()

	s.running = true
	s.logger.Info("元数据服务器启动成功")

	return nil
}

// Stop 停止服务器
func (s *MetadataServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 停止HTTP服务器
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP服务器关闭失败: %v", err)
	}

	// 停止集群服务
	if err := s.cluster.Stop(ctx); err != nil {
		s.logger.Error("集群服务关闭失败: %v", err)
	}

	// 关闭元数据存储
	if err := s.metaStore.Close(); err != nil {
		s.logger.Error("元数据存储关闭失败: %v", err)
	}

	s.running = false
	s.logger.Info("元数据服务器已停止")

	return nil
}

// IsRunning 检查服务器是否正在运行
func (s *MetadataServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// setupRoutes 设置HTTP路由
func (s *MetadataServer) setupRoutes(httpServer *nethttp.Server) {
	// 根路径 - 健康检查
	httpServer.GET("/", s.handleHealthCheck)

	// API版本前缀
	apiV1 := "/api/v1"

	// 元数据操作
	httpServer.GET(apiV1+"/files/*path", s.handleGetFileInfo)
	httpServer.POST(apiV1+"/files/*path", s.handleCreateFile)
	httpServer.PUT(apiV1+"/files/*path", s.handleUpdateFile)
	httpServer.DELETE(apiV1+"/files/*path", s.handleDeleteFile)
	httpServer.GET(apiV1+"/dirs/*path", s.handleListDirectory)
	httpServer.POST(apiV1+"/dirs/*path", s.handleCreateDirectory)
	httpServer.DELETE(apiV1+"/dirs/*path", s.handleDeleteDirectory)

	// 集群操作
	httpServer.GET(apiV1+"/cluster/nodes", s.handleListNodes)
	httpServer.GET(apiV1+"/cluster/nodes/:id", s.handleGetNodeInfo)
	httpServer.GET(apiV1+"/cluster/leader", s.handleGetLeader)

	// 管理操作
	httpServer.GET(apiV1+"/admin/status", s.handleServerStatus)
}
