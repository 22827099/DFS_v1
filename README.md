# 分布式文件系统 (DFS_v1)

这是一个高性能、高可用的分布式文件系统实现，提供可靠的文件存储和访问服务，支持数据冗余和容错机制。

## 项目概述

DFS_v1 是一个完整的分布式文件系统解决方案，具有以下特点：

- **高可用性**：通过数据复制和容错机制确保服务持续可用
- **可扩展性**：支持集群动态扩展，适应不断增长的存储需求
- **一致性保证**：采用分布式共识算法确保数据一致性 // TODO ：Raft共识算法
- **安全机制**：提供完整的身份验证和访问控制

## 系统架构

系统采用主从架构设计：

- **元数据服务器**：负责元数据管理、命名空间管理和集群协调
- **数据服务器**：负责实际文件数据的存储和读写
- **客户端**：提供用户交互接口，包括命令行工具和SDK

## 主要功能

- 文件的分布式存储和读写
- 文件的分片、复制和容错
- 元数据的一致性管理
- 分布式一致性保证
- 客户端访问接口（命令行和API）
- 集群自动负载均衡
- 高性能数据传输
- 完善的安全认证机制

## 目录结构
```txt
DFS_v1/
├── cmd/                          # 各组件的入口点
│   ├── client/                   # 客户端命令行工具
│   ├── dataserver/               # 数据服务器主程序
│   └── metaserver/               # 元数据服务器主程序
|
├── config/                       # 各组件配置文件
|
├── pkg/                          # 可被外部导入的包
│   ├── api/                      # 公开 API 定义
│   ├── client/                   # 客户端库
│   └── protocol/                 # 协议定义
│
├── internal/                     # 内部实现，不导出
│   ├── client/                   # 客户端内部实现
│   ├── dataserver/               # 数据服务器实现
│   └── metaserver/               # 元数据服务器实现
│
├── common/                       # 公共代码和工具
│   ├── config/                   # 配置处理
│   ├── errors/                   # 错误处理
│   ├── logging/                  # 日志功能
│   ├── metrics/                  # 监控与指标
│   ├── network/                  # 网络通信
│   ├── consensus/                # 分布式一致性
│   ├── security/                 # 安全与认证
│   ├── concurrency/              # 并发控制
│   └── utils/                    # 工具函数
│
├── scripts/                      # 脚本工具
│   ├── build/                    # 构建脚本
│   ├── deploy/                   # 部署脚本
│   └── bench/                    # 性能测试脚本
│
├── test/                         # 测试代码
│   ├── unit/                     # 单元测试
│   ├── integration/              # 集成测试
│   ├── e2e/                      # 端到端测试
│   ├── performance/              # 性能测试
│   └── stress/                   # 压力测试
│
├── examples/                     # 使用示例
│   ├── basic/                    # 基本使用
│   ├── advanced/                 # 高级功能
│   └── deployment/               # 部署示例
│
└── docs/                         # 文档
    ├── architecture/             # 架构文档
    ├── api/                      # API 文档
    ├── protocols/                # 协议文档
    ├── design/                   # 设计文档
    ├── user-guide/               # 用户指南
    └── development/              # 开发文档
```

## 开发环境要求

- Go 1.16+ 开发环境
- Git 版本控制
- Make 构建工具
- Docker 和 Kubernetes (用于容器化部署)

## 编译与运行
### 本地编译
```bash
# 编译系统
cd scripts/build
./build.sh

# 启动元数据服务器
./scripts/deploy/start_metaserver.sh

# 启动数据服务器
./scripts/deploy/start_dataserver.sh

# 运行客户端
./scripts/deploy/start_client.sh
```
### 容器部署
```bash
# Docker 部署
cd scripts/deploy/docker
./docker-build.sh
docker-compose up

# Kubernetes 部署
cd scripts/deploy/kubernetes
./deploy.sh
```

## 配置说明

系统配置文件位于 `config/` 目录下，主要包括：

- `metaserver_config.json`: 元数据服务器配置
- `dataserver_config.json`: 数据服务器配置
- `client_config.json`: 客户端配置
- `replication_config.json`: 复制策略配置
- `security_config.json`: 安全配置

## 测试
```bash
# 运行单元测试
go test ./test/unit/...

# 运行集成测试
go test ./test/integration/...

# 运行端到端测试
./test/e2e/run_e2e_tests.sh

# 运行性能测试
./test/performance/run_performance_tests.sh

# 运行压力测试
./test/stress/run_stress_tests.sh
```

## 文档
详细文档位于 `/docs` 目录：

- 架构概述
- API参考
- 部署指南
- 用户指南
- 开发指南

## 示例

examples 目录包含各种使用示例：

- 基本文件操作示例
- 高级功能示例（并发访问、容错等）
- 不同环境的部署示例

## 联系方式

[2282706227@qq.com]
