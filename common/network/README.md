# 网络通信模块

本目录提供分布式文件系统中的网络通信功能。模块设计采用简单封装方式，专注于提供核心功能，避免过度设计。

## 模块组成

- **http/** - HTTP客户端和服务器封装
  - 简单易用的HTTP客户端，支持JSON请求和响应处理
  - 基于gorilla/mux的HTTP服务器，提供轻量级路由功能
  
- **transport/** - 网络传输抽象层
  - 统一的传输接口定义
  - 支持不同传输协议的抽象

## 使用示例

### HTTP客户端

```go
// 创建HTTP客户端
client := http.NewClient("http://localhost:8080")

// 发送GET请求并解析JSON响应
var result map[string]interface{}
err := client.GetJSON(context.Background(), "/api/status", &result)

// 创建HTTP服务器
server := http.NewServer(":8080")

// 注册路由处理函数
server.GET("/api/files", func(w http.ResponseWriter, r *http.Request) {
    files := []string{"file1.txt", "file2.txt"}
    http.RespondJSON(w, http.StatusOK, http.SuccessResponse(files))
})

// 启动服务器
if err := server.Start(); err != nil {
    log.Fatal(err)
}