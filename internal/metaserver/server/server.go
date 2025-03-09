package server

import (
    "context"
    "fmt"
    "time"

    "github.com/22827099/DFS_v1/common/logging"
    "github.com/22827099/DFS_v1/common/network/http"
    "github.com/22827099/DFS_v1/internal/metaserver/config"
    "github.com/22827099/DFS_v1/internal/metaserver/core"
    "github.com/22827099/DFS_v1/internal/metaserver/server/handler"
)

// MetadataServer 表示元数据服务器实例
type MetadataServer struct {
    config     *config.Config
    logger     logging.Logger
    httpServer *http.Server
    core       *core.MetaCore
    handlers   *handler.Handlers
}

// NewServer 创建新的元数据服务器实例
func NewServer(cfg *config.Config, logger logging.Logger) (*MetadataServer, error) {
    // 初始化核心组件
    metaCore, err := core.NewMetaCore(cfg, logger)
    if err != nil {
        return nil, fmt.Errorf("初始化元数据核心组件失败: %w", err)
    }

    // 创建HTTP服务器
    serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)
    server := http.NewServer(serverAddr)
    server.SetLogger(logger)
    server.SetReadTimeout(cfg.Server.ReadTimeout)
    server.SetWriteTimeout(cfg.Server.WriteTimeout)
    server.SetIdleTimeout(time.Minute)

    // 创建请求处理器
    handlers := handler.NewHandlers(metaCore, logger)

    metaServer := &MetadataServer{
        config:     cfg,
        logger:     logger,
        httpServer: server,
        core:       metaCore,
        handlers:   handlers,
    }

    // 注册中间件
    metaServer.registerMiddlewares()
    
    // 注册API路由
    metaServer.registerRoutes()

    return metaServer, nil
}

// registerMiddlewares 注册HTTP中间件
func (s *MetadataServer) registerMiddlewares() {
    // 添加请求日志中间件
    s.httpServer.Use(http.LoggingMiddleware(s.logger))
    
    // 添加恢复中间件，防止panic导致服务器崩溃
    s.httpServer.Use(http.RecoveryMiddleware(s.logger))
    
    // 添加请求跟踪中间件
    s.httpServer.Use(http.RequestIDMiddleware())
    
    // 根据配置添加CORS中间件
    if s.config.Server.EnableCORS {
        s.httpServer.Use(http.CORSMiddleware(s.config.Server.AllowOrigins))
    }
    
    // 添加基本认证中间件（如果启用）
    if s.config.Security.EnableAuth {
        s.httpServer.Use(http.AuthMiddleware(s.handlers.AuthHandler))
    }
}

// registerRoutes 注册所有API路由
func (s *MetadataServer) registerRoutes() {
    // API根路径组
    api := s.httpServer.Router().Group("/api/v1")

    // 文件和目录操作
    fs := api.Group("/fs")
    fs.GET("/{path:.*}", http.Adapt(s.handlers.GetMetadata))
    fs.PUT("/{path:.*}", http.Adapt(s.handlers.CreateOrUpdateMetadata))
    fs.DELETE("/{path:.*}", http.Adapt(s.handlers.DeleteMetadata))
    fs.POST("/{path:.*}/move", http.Adapt(s.handlers.MoveMetadata))
    fs.POST("/{path:.*}/copy", http.Adapt(s.handlers.CopyMetadata))
    fs.GET("/{path:.*}/list", http.Adapt(s.handlers.ListDirectory))
    
    // 用户和权限管理
    users := api.Group("/users")
    users.POST("", http.Adapt(s.handlers.CreateUser))
    users.GET("/{userId}", http.Adapt(s.handlers.GetUser))
    users.PUT("/{userId}", http.Adapt(s.handlers.UpdateUser))
    
    // 文件系统操作
    api.POST("/operations/batch", http.Adapt(s.handlers.BatchOperation))

    // 集群管理API
    cluster := api.Group("/cluster")
    cluster.GET("/status", http.Adapt(s.handlers.GetClusterStatus))
    cluster.GET("/nodes", http.Adapt(s.handlers.ListClusterNodes))
    cluster.POST("/rebalance", http.Adapt(s.handlers.StartRebalance))
    
    // 系统管理API
    admin := api.Group("/admin")
    admin.GET("/stats", http.Adapt(s.handlers.GetSystemStats))
    admin.GET("/health", http.Adapt(s.handlers.HealthCheck))
    
    // 系统状态API（无需认证）
    s.httpServer.GET("/status", http.Adapt(s.handlers.SystemStatus))
}

// Start 启动服务器
func (s *MetadataServer) Start() error {
    s.logger.Info("元数据服务器启动于 %s:%d", s.config.Server.Address, s.config.Server.Port)

    // 启动核心服务
    if err := s.core.Start(); err != nil {
        return fmt.Errorf("启动核心服务失败", err)
    }

    // 启动HTTP服务器
    if s.config.Security.EnableTLS {
        s.logger.Info("启用TLS加密")
        return s.httpServer.StartTLS(s.config.Security.CertFile, s.config.Security.KeyFile)
    }
    return s.httpServer.Start()
}

// Stop 停止服务器
func (s *MetadataServer) Stop(ctx context.Context) error {
    s.logger.Info("正在停止元数据服务器...")
    
    // 停止HTTP服务器
    if err := s.httpServer.Stop(ctx); err != nil {
        s.logger.Error("停止HTTP服务器错误: %v", err)
    }

    // 停止核心服务
    if err := s.core.Stop(ctx); err != nil {
        return fmt.Errorf("停止核心服务失败", err)
    }
    
    s.logger.Info("元数据服务器已成功停止")
    return nil
}