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
	"github.com/22827099/DFS_v1/common/metrics"
	nethttp "github.com/22827099/DFS_v1/common/network/http"
	"github.com/22827099/DFS_v1/internal/metaserver/core"
	metaconfig "github.com/22827099/DFS_v1/internal/metaserver/config"
	"github.com/22827099/DFS_v1/internal/metaserver/core/cluster"
	"github.com/22827099/DFS_v1/internal/metaserver/core/metadata"
	"github.com/22827099/DFS_v1/internal/metaserver/server/middleware"
	"github.com/22827099/DFS_v1/internal/metaserver/server/api/v1"
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
	metricsCollector metrics.Collector
    metaCore         *core.MetaCore       // 添加这个字段
	authService      middleware.AuthService       // 添加认证服务
    txManager        middleware.TransactionManager // 添加事务管理器
}

// ServerOption 允许配置服务器的选项函数
type ServerOption func(*MetadataServer)

// NewServer 创建新的元数据服务器
func NewServer(cfg *config.SystemConfig, options ...ServerOption) (*MetadataServer, error) {
	if cfg == nil {
		return nil, errors.New(errors.InvalidArgument, "配置不能为空")
	}

    // 初始化日志
    logger := logging.NewLogger()
    
    // 初始化 HTTP 服务器
    httpServer := nethttp.NewServer(fmt.Sprintf("%s:%d", "localhost", 8080))

	// // 初始化认证服务 TODO: #2 添加认证服务
	// authService := middleware.Auth(/* 必要参数 */)

	// // 初始化事务管理器 TODO: #3 添加事务管理器
	// txManager := db.NewTransactionManager(/* 必要参数 */)
    
    // 转换为元数据服务器配置
    metaCfg := &metaconfig.Config{
		Database: metaconfig.DatabaseConfig{},
		Cluster:  metaconfig.ClusterConfig{},
    }
    
    // 初始化元数据核心
    metaCore, err := core.NewMetaCore(metaCfg, logger)
    if err != nil {
        return nil, errors.Wrap(err, errors.Internal, "failed to initialize meta core")
    }
    
    // 初始化指标收集器
    metricsCollector := metrics.NewCollector("metaserver")
    
    // 创建服务器实例
    server := &MetadataServer{
        config:           cfg,
        logger:           logger,
        httpServer:       httpServer,
        metaCore:         metaCore,
        metricsCollector: metricsCollector,
        running:          false,
		// authService:      authService,  // 注释掉
        // txManager:        txManager,    // 注释掉
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
    // 注册中间件
    httpServer.Use(nethttp.RequestIDMiddleware())
    httpServer.Use(nethttp.LoggingMiddleware(s.logger))
    httpServer.Use(nethttp.RecoveryMiddleware(s.logger))
    httpServer.Use(middleware.Metrics(s.metricsCollector))
    httpServer.Use(middleware.RateLimit(100, 1*time.Second))
    
    // 为需要认证的路由组添加认证中间件
    apiRouter := httpServer.Group("/api/v1")
    apiRouter.Use(middleware.Auth(s.authService))
    apiRouter.Use(middleware.Transaction(s.txManager))
    
    // 创建并注册API处理器
    filesAPI := v1.NewFilesAPI(s.metaStore)
    dirsAPI := v1.NewDirectoriesAPI(s.metaStore)
    clusterAPI := v1.NewClusterAPI(s.cluster)
    adminAPI := v1.NewAdminAPI(s.config, s.cluster)
    
    // 注册路由
	filesAPI.RegisterRoutes(apiRouter)
	dirsAPI.RegisterRoutes(apiRouter)
	clusterAPI.RegisterRoutes(apiRouter)
	adminAPI.RegisterRoutes(apiRouter)
    
    // 公开的健康检查端点
    httpServer.GET("/health", adminAPI.HealthCheck)
}
