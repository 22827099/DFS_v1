package main

import (
	"context"
	"fmt"
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

	// 等待服务器启动
	time.Sleep(500 * time.Millisecond)

	// 测试客户端功能
	testClientFunctionality(logger)

	// 停止服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Stop(ctx)

	fmt.Println("所有测试通过！")
}

func testClientFunctionality(logger logging.Logger) {
	// 创建HTTP客户端
	client := http.NewClient("http://localhost:8080",
		http.WithTimeout(5*time.Second),
		http.WithRetryPolicy(http.DefaultRetryPolicy()),
	)

	// 测试1: 基本GET请求
	var helloResult map[string]interface{}
	err := client.GetJSON(context.Background(), "/hello", nil, &helloResult)
	if err != nil {
		logger.Error("Hello请求错误: %v", err)
		panic("测试失败")
	}
	logger.Info("Hello响应: %v", helloResult)

	// 测试2: 路径参数
	var userResult map[string]interface{}
	err = client.GetJSON(context.Background(), "/users/123", nil, &userResult)
	if err != nil {
		logger.Error("Users请求错误: %v", err)
		panic("测试失败")
	}
	logger.Info("Users响应: %v", userResult)

	// 测试3: API分组
	var statusResult map[string]interface{}
	err = client.GetJSON(context.Background(), "/api/status", nil, &statusResult)
	if err != nil {
		logger.Error("Status请求错误: %v", err)
		panic("测试失败")
	}
	logger.Info("Status响应: %v", statusResult)
}
