# 元数据服务器实现

此目录包含分布式文件系统中元数据服务器的实现，负责管理文件系统的命名空间、元数据和集群协调。

## 目录结构

- **config/** - 配置处理
  - 加载和解析配置文件
  - 提供配置默认值和验证
  - 支持动态配置更新

- **server/** - 服务器实现
  - 提供HTTP API接口
  - 请求路由和处理
  - API认证和授权
  - 中间件支持（日志、恢复、CORS等）
  - 处理器实现（文件操作、集群管理、系统状态等）
  - 错误处理和响应格式化

- **core/** - 核心功能模块
  - **database/** - 数据库访问层
  - **metadata/** - 元数据管理
  - **cluster/** - 集群管理与协调

## API概览

元数据服务器提供以下主要API:

### 文件系统操作
- `GET /api/v1/fs/{path}` - 获取路径元数据
- `PUT /api/v1/fs/{path}` - 创建/更新文件或目录
- `DELETE /api/v1/fs/{path}` - 删除文件或目录
- `GET /api/v1/fs/{path}/list` - 列出目录内容（支持排序）
- `POST /api/v1/fs/{path}/move` - 移动文件或目录
- `POST /api/v1/fs/{path}/copy` - 复制文件或目录
- `POST /api/v1/operations/batch` - 批量操作

### 集群管理
- `GET /api/v1/cluster/status` - 获取集群状态
- `GET /api/v1/cluster/nodes` - 列出集群节点
- `POST /api/v1/cluster/rebalance` - 触发负载均衡

### 系统管理
- `GET /api/v1/admin/stats` - 获取系统统计信息
- `GET /api/v1/admin/health` - 健康检查
- `GET /status` - 系统状态（无需认证）

## 实现特性

- **高可用性**：支持多节点部署，自动故障转移
- **一致性保证**：使用分布式共识算法保证数据一致性
- **可扩展性**：支持动态增减节点，自动数据再平衡
- **安全机制**：TLS加密、身份验证和授权

## 使用示例

启动元数据服务器:

```bash
./cmd/metaserver/metaserver -config config/metaserver_config.json
```

## 开发指南

扩展服务器功能:

1. 在 `server/handler` 目录添加新的处理函数
2. 在 `server.go` 的 `registerRoutes()` 方法中注册新的路由
3. 如需添加中间件，在 `registerMiddlewares()` 方法中配置
