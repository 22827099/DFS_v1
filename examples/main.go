package main

import (
    "context"
    "time"
    
    "github.com/22827099/DFS_v1/common/logging"
    "github.com/22827099/DFS_v1/common/network/http"
)

func main() {
    // 创建一个日志记录器
    logger := logging.NewLogger()
    
    // 创建服务器
    server := http.NewServer(":8080", 
        http.WithServerLogger(logger),
        http.WithReadTimeout(5*time.Second),
        http.WithWriteTimeout(10*time.Second),
    )
    
    // 添加路由
    server.GET("/hello", func(c *http.Context) {
        http.WriteSuccess(c.Response, map[string]string{"message": "Hello, World!"})
    })
    
    // 添加带参数的路由
    server.GET("/users/{id}", func(c *http.Context) {
        userID := c.Params["id"]
        http.WriteSuccess(c.Response, map[string]string{"user_id": userID})
    })
    
    // 创建API路由组
    apiGroup := server.Router().Group("/api")
    apiGroup.GET("/status", func(c *http.Context) {
        http.WriteSuccess(c.Response, map[string]string{"status": "ok"})
    })
    
    // 启动服务器
    go func() {
        logger.Info("启动HTTP服务器...")
        if err := server.Start(); err != nil {
            logger.Error("服务器错误: %v", err)
        }
    }()
    
    // 创建HTTP客户端
    client := http.NewClient("http://localhost:8080",
        http.WithTimeout(5*time.Second),
        http.WithRetryPolicy(http.DefaultRetryPolicy()),
    )
    
    // 等待服务器启动
    time.Sleep(1 * time.Second)
    
    // 发送请求
    var result map[string]interface{}
    err := client.GetJSON(context.Background(), "/hello", nil, &result)
    if err != nil {
        logger.Error("请求错误: %v", err)
    } else {
        logger.Info("响应: %v", result)
    }
    
    // 停止服务器
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    server.Stop(ctx)
}