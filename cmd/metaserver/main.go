package main

import (
	// "context"
	// "flag"
	// "os"
	// "os/signal"
	// "syscall"

	// "github.com/22827099/DFS_v1/common/logging"
	// "github.com/22827099/DFS_v1/internal/metaserver/config"
	// "github.com/22827099/DFS_v1/internal/metaserver/server"
)

// 元数据服务器入口点

func main() {
	// // 1. 解析命令行参数
	// configPath := flag.String("config", "config/metaserver_config.json", "配置文件路径")
	// flag.Parse()

	// // 2. 初始化日志
	// logger := logging.NewLogger()
	// logger.Info("元数据服务器正在启动...")

	// // 3. 加载配置
	// cfg, err := config.LoadConfig(*configPath)
	// if err != nil {
	// 	logger.Fatal("加载配置失败: %v", err)
	// }

	// // 4. 创建并初始化服务器实例
	// metaServer, err := server.NewServer(cfg, logger)
	// if err != nil {
	// 	logger.Fatal("初始化服务器失败: %v", err)
	// }

	// // 5. 启动服务器（非阻塞）
	// if err := metaServer.Start(); err != nil {
	// 	logger.Fatal("启动服务器失败: %v", err)
	// }

	// // 6. 等待中断信号
	// signalChan := make(chan os.Signal, 1)
	// signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	// <-signalChan

	// // 7. 优雅关闭服务器
	// logger.Info("正在关闭服务器...")
	// ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	// defer cancel()
	// if err := metaServer.Stop(ctx); err != nil {
	// 	logger.Error("服务器关闭出错: %v", err)
	// }
}
