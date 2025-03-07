
# 集成测试

此目录包含系统组件间的集成测试：
- client_server_test.go - 客户端与服务器交互测试
- dataserver_metaserver_test.go - 数据服务器与元数据服务器交互测试
- cluster_test.go - 集群功能测试

运行集成测试：
```bash
go test ./test/integration/...

