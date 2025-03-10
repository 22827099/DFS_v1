package v1

import (
	"net/http"
	"runtime"
	"time"

	"github.com/22827099/DFS_v1/common/config"
	"github.com/22827099/DFS_v1/internal/metaserver/core/cluster"
	nethttp "github.com/22827099/DFS_v1/common/network/http"
	"github.com/shirou/gopsutil/cpu"
	"github.com/22827099/DFS_v1/internal/metaserver/server/api"
)

// AdminAPI 处理管理相关的API请求
type AdminAPI struct {
	config  *config.SystemConfig
	cluster cluster.Manager
	startTime time.Time      // 服务启动时间
    // connMgr   *ConnectionManager // TODO: #1 添加连接管理器
}

// 获取活跃连接数
func (a *AdminAPI) getActiveConnections() int {
    // if a.connMgr != nil {
    //     return a.connMgr.GetActiveConnectionCount()
    // }
    return 0
}

// NewAdminAPI 创建管理API处理器
func NewAdminAPI(config *config.SystemConfig, cluster cluster.Manager) *AdminAPI {
    return &AdminAPI{
        config:    config,
        cluster:   cluster,
        startTime: time.Now(),
    }
}

// RegisterRoutes 注册管理相关路由
func (a *AdminAPI) RegisterRoutes(router nethttp.RouteGroup) {
	router.GET("/health", a.HealthCheck)
	router.GET("/status", a.ServerStatus)
}

// HealthCheck 处理健康检查请求
func (a *AdminAPI) HealthCheck(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "running",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
	}

	api.RespondSuccess(w, r, http.StatusOK, status)
}

// 以下是辅助函数，用于获取系统资源使用情况
func getMemoryUsage() float64 {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    return float64(m.Alloc) / 1024 / 1024 // 返回MB
}

func getCPUUsage() float64 {
	// 获取CPU使用率，采样间隔为100毫秒
	// 参数false表示获取整体CPU使用率，而不是每个CPU核心的使用率
	percent, err := cpu.Percent(100*time.Millisecond, false)
	if err != nil {
		// 如果发生错误，返回0
		// 在生产环境中，应考虑更完善的错误处理
		return 0.0
	}
	
	// cpu.Percent返回一个切片，当percpu=false时，只包含整体使用率
	if len(percent) > 0 {
		return percent[0]
	}
	
	return 0.0
}

func getDiskUsage() map[string]float64 {
    // 需要使用系统调用或第三方库获取磁盘使用情况
    return map[string]float64{
        "total_gb":     100.0, // 示例
        "used_gb":      50.0,  // 示例
        "percent_used": 50.0,  // 示例
    }
}

// ServerStatus 获取服务器状态
func (a *AdminAPI) ServerStatus(w http.ResponseWriter, r *http.Request) {
	isLeader := a.cluster.IsLeader()
	
	status := map[string]interface{}{
		"id":          a.config.NodeID,                		// 节点ID
		"uptime":      time.Since(a.startTime).String(), 	// 服务运行时间
		"is_leader":   isLeader,                       		// 是否为集群领导节点
		// "connections": a.getActiveConnections(),       		// 活跃连接数
		"version":     a.config.Version,               		// 服务版本号
		"system_info": map[string]interface{}{
			"memory_usage": getMemoryUsage(),         		// 内存使用量(MB)
			"cpu_usage":    getCPUUsage(),            		// CPU使用率(百分比)
			"disk_usage":   getDiskUsage(),          		// 磁盘使用情况
			"goroutines":   runtime.NumGoroutine(),  		// 当前goroutine数量
		},
		"cluster_info": map[string]interface{}{
			"node_count":    a.cluster.GetNodeCount(),       	// 集群节点总数
			"healthy_nodes": a.cluster.GetHealthyNodeCount(), 	// 健康节点数量
			"leader_id":     a.cluster.GetCurrentLeader(),        	// 当前领导节点ID
			"last_election": a.cluster.LastElectionTime().Format(time.RFC3339), // 最后一次选举时间
		},
	}

    api.RespondSuccess(w, r, http.StatusOK, status)
}
