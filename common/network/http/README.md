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