package v1

import (
	"net/http"

	"github.com/22827099/DFS_v1/internal/metaserver/core/cluster"
	nethttp "github.com/22827099/DFS_v1/common/network/http"
)

// ClusterAPI 处理集群相关的API请求
type ClusterAPI struct {
	cluster cluster.Manager
}

// NewClusterAPI 创建集群API处理器
func NewClusterAPI(cluster cluster.Manager) *ClusterAPI {
	return &ClusterAPI{
		cluster: cluster,
	}
}

// RegisterRoutes 注册集群相关路由
func (c *ClusterAPI) RegisterRoutes(router nethttp.RouteGroup) {
	router.GET("/nodes", c.ListNodes)
	router.GET("/nodes/{id}", c.GetNodeInfo)
	router.GET("/leader", c.GetLeader)
	router.POST("/rebalance", c.TriggerRebalance)
	router.GET("/rebalance/status", c.GetRebalanceStatus)
}

// ListNodes 列出集群节点
func (c *ClusterAPI) ListNodes(w http.ResponseWriter, r *http.Request) {
	// 从原来的 handleListNodes 转换而来
	// ...
}

// GetNodeInfo 获取节点信息
func (c *ClusterAPI) GetNodeInfo(w http.ResponseWriter, r *http.Request) {
	// 从原来的 handleGetNodeInfo 转换而来
	// ...
}

// GetLeader 获取当前集群领导者信息
func (c *ClusterAPI) GetLeader(w http.ResponseWriter, r *http.Request) {
	// 从原来的 handleGetLeader 转换而来
	// ...
}

// 可以添加其他集群管理功能...
// TriggerRebalance 触发数据均衡
func (c *ClusterAPI) TriggerRebalance(w http.ResponseWriter, r *http.Request) {
	// 从原来的 handleTriggerRebalance 转换而来
	// ...
}

// GetRebalanceStatus 获取数据均衡状态
func (c *ClusterAPI) GetRebalanceStatus(w http.ResponseWriter, r *http.Request) {
	// 从原来的 handleGetRebalanceStatus 转换而来
	// ...
}
