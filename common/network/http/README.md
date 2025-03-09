# HTTP 基础封装

此目录提供对 HTTP 通信的轻量级封装，专注于简单易用，避免过度设计：

## 主要组件

- **客户端 (client.go)**:
  - 基于标准库 `net/http` 的简单封装
  - 支持 JSON 请求和响应处理
  - 提供简洁的 API 接口：`Get`、`Post`、`GetJSON`、`PostJSON`

- **服务器 (server.go)**:
  - 基于 `gorilla/mux` 的轻量级路由
  - 提供简单的路由注册方法：`GET`、`POST`、`PUT`、`DELETE`
  - 包含基础响应处理：`RespondJSON`、`RespondError`

- **中间件 (middleware.go)**
  - 提供可组合的HTTP请求处理链
  - 包含多种实用中间件：
    - 日志中间件 - 记录请求路径、方法、状态码和响应时间
    - 恢复中间件 - 捕获处理过程中的panic，防止服务器崩溃
    - 请求ID中间件 - 为每个请求分配唯一标识符，便于追踪与调试
    - CORS中间件 - 处理跨域资源共享，支持配置允许的来源
    - 认证中间件 - 基于HTTP Basic认证，保护API资源
  
## 使用示例

```go
// 客户端示例
client := http.NewClient("http://localhost:8080")
var result map[string]interface{}
err := client.GetJSON(context.Background(), "/api/status", &result)

// 服务器示例
server := http.NewServer(":8080")
server.GET("/api/status", func(w http.ResponseWriter, r *http.Request) {
    http.RespondJSON(w, 200, map[string]string{"status": "ok"})
})
server.Start()

// 中间件示例
logger := logging.NewLogger("server")
server := http.NewServer(":8080")

// 添加中间件
server.Use(http.LoggingMiddleware(logger))
server.Use(http.RecoveryMiddleware(logger))
server.Use(http.RequestIDMiddleware())
server.Use(http.CORSMiddleware([]string{"*"}))

// 添加认证中间件
authFunc := func(username, password string) bool {
    return username == "admin" && password == "secret"
}
server.Use(http.AuthMiddleware(authFunc))

// 使用请求ID
server.GET("/api/data", func(w http.ResponseWriter, r *http.Request) {
    requestID := http.GetRequestID(r.Context())
    logger.Info("处理请求 ID: %s", requestID)
    // 处理请求...
})