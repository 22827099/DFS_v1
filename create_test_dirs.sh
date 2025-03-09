#!/bin/bash

# 创建测试目录结构的脚本
echo "开始创建测试目录结构..."

# 创建单元测试目录
mkdir -p test/unit/metaserver/core
mkdir -p test/unit/metaserver/store
mkdir -p test/unit/cluster/rebalance
mkdir -p test/unit/cluster/manager
mkdir -p test/unit/consensus
mkdir -p test/unit/common/config
mkdir -p test/unit/common/network
mkdir -p test/unit/common/concurrency

# 创建集成测试目录
mkdir -p test/integration/metaserver_cluster
mkdir -p test/integration/meta_data_servers
mkdir -p test/integration/client_server

# 创建其他测试目录
mkdir -p test/e2e
mkdir -p test/performance
mkdir -p test/fixtures

echo "测试目录结构创建完成！"

# 列出创建的目录结构
find test -type d | sort