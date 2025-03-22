package metaserver_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/22827099/DFS_v1/common/config"
	"github.com/22827099/DFS_v1/internal/metaserver/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClusterManager 测试集群管理功能
func TestClusterManager(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过耗时的集群测试")
	}

	// 创建3个节点的集群配置
	clusterSize := 3
	servers := make([]*server.Server, clusterSize)
	configs := make([]*config.SystemConfig, clusterSize)
	baseURLs := make([]string, clusterSize)

	// 基础端口号
	basePort := 19000

	// 准备所有节点的配置
	for i := 0; i < clusterSize; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		configs[i] = &config.SystemConfig{
			NodeID: nodeID,
			Server: config.ServerConfig{
				Host: "localhost",
				Port: basePort + i,
			},
			Cluster: config.ClusterConfig{
				Peers: []string{
					fmt.Sprintf("localhost:%d", basePort),
					fmt.Sprintf("localhost:%d", basePort+1),
					fmt.Sprintf("localhost:%d", basePort+2),
				},
				ElectionTimeout:  2 * time.Second,
				HeartbeatTimeout: 1 * time.Second,
			},
		}
		baseURLs[i] = fmt.Sprintf("http://localhost:%d", basePort+i)
	}

	// 测试完成后清理所有服务器
	defer func() {
		for _, s := range servers {
			if s != nil {
				s.Stop()
			}
		}
	}()

	t.Run("NodeJoinTest", func(t *testing.T) {
		// 先启动一个节点作为初始节点
		servers[0], _ = server.NewServer(configs[0])
		err := servers[0].Start()
		require.NoError(t, err)

		// 给第一个节点一些时间启动
		time.Sleep(1 * time.Second)

		// 获取初始节点的集群信息
		resp, err := http.Get(baseURLs[0] + "/api/v1/cluster/nodes")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var initialNodes map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&initialNodes)
		require.NoError(t, err)

		// 验证初始只有一个节点
		nodesList, ok := initialNodes["nodes"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 1, len(nodesList), "初始应该只有一个节点")

		// 启动第二个节点并加入集群
		servers[1], _ = server.NewServer(configs[1])
		err = servers[1].Start()
		require.NoError(t, err)

		// 等待节点加入
		time.Sleep(3 * time.Second)

		// 验证集群现在有两个节点
		resp, err = http.Get(baseURLs[0] + "/api/v1/cluster/nodes")
		require.NoError(t, err)
		defer resp.Body.Close()

		var updatedNodes map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&updatedNodes)
		require.NoError(t, err)

		nodesList, ok = updatedNodes["nodes"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 2, len(nodesList), "集群应该有两个节点")

		// 验证第二个节点视角
		resp, err = http.Get(baseURLs[1] + "/api/v1/cluster/nodes")
		require.NoError(t, err)
		defer resp.Body.Close()

		var node2View map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&node2View)
		require.NoError(t, err)

		node2List, ok := node2View["nodes"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 2, len(node2List), "第二个节点也应该看到两个节点")
	})

	t.Run("NodeLeaveTest", func(t *testing.T) {
		// 使用现有的两个节点

		// 停止第二个节点
		err := servers[1].Stop()
		require.NoError(t, err)
		servers[1] = nil

		// 等待心跳超时检测到节点离开
		time.Sleep(5 * time.Second)

		// 验证集群现在只有一个活跃节点
		resp, err := http.Get(baseURLs[0] + "/api/v1/cluster/nodes")
		require.NoError(t, err)
		defer resp.Body.Close()

		var clusterState map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&clusterState)
		require.NoError(t, err)

		nodesList, ok := clusterState["nodes"].([]interface{})
		require.True(t, ok)

		// 计算健康状态的节点数
		healthyCount := 0
		for _, node := range nodesList {
			nodeInfo := node.(map[string]interface{})
			if nodeInfo["status"] == "healthy" {
				healthyCount++
			}
		}

		assert.Equal(t, 1, healthyCount, "应该只有一个健康状态的节点")
	})

	t.Run("HeartbeatTest", func(t *testing.T) {
		// 启动第二个节点
		servers[1], _ = server.NewServer(configs[1])
		err := servers[1].Start()
		require.NoError(t, err)

		// 等待心跳建立
		time.Sleep(3 * time.Second)

		// 验证两个节点都是健康状态
		resp, err := http.Get(baseURLs[0] + "/api/v1/cluster/nodes")
		require.NoError(t, err)
		defer resp.Body.Close()

		var clusterState map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&clusterState)
		require.NoError(t, err)

		nodesList, ok := clusterState["nodes"].([]interface{})
		require.True(t, ok)

		// 计算健康状态的节点数
		healthyCount := 0
		for _, node := range nodesList {
			nodeInfo := node.(map[string]interface{})
			if nodeInfo["status"] == "healthy" {
				healthyCount++
			}
		}

		assert.Equal(t, 2, healthyCount, "应该有两个健康状态的节点")

		// 获取心跳统计信息
		resp, err = http.Get(baseURLs[0] + "/api/v1/cluster/stats")
		require.NoError(t, err)
		defer resp.Body.Close()

		var stats map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&stats)
		require.NoError(t, err)

		// 验证心跳统计包含关键指标
		assert.Contains(t, stats, "heartbeat_sent")
		assert.Contains(t, stats, "heartbeat_received")
		assert.Contains(t, stats, "last_heartbeat_time")
	})

	t.Run("LeaderElectionTest", func(t *testing.T) {
		// 启动第三个节点，形成完整的三节点集群
		servers[2], _ = server.NewServer(configs[2])
		err := servers[2].Start()
		require.NoError(t, err)

		// 等待选举完成
		time.Sleep(5 * time.Second)

		// 获取当前领导者
		resp, err := http.Get(baseURLs[0] + "/api/v1/cluster/leader")
		require.NoError(t, err)
		defer resp.Body.Close()

		var leaderInfo map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&leaderInfo)
		require.NoError(t, err)

		// 记录当前领导者ID
		require.Contains(t, leaderInfo, "node_id")
		currentLeaderID := leaderInfo["node_id"].(string)
		t.Logf("当前领导者: %s", currentLeaderID)

		// 确定领导者节点索引
		leaderIdx := -1
		for i, cfg := range configs {
			if cfg.NodeID == currentLeaderID {
				leaderIdx = i
				break
			}
		}
		require.NotEqual(t, -1, leaderIdx, "找不到领导者节点索引")

		// 停止当前领导者节点触发重新选举
		err = servers[leaderIdx].Stop()
		require.NoError(t, err)
		servers[leaderIdx] = nil

		// 等待新的选举完成
		time.Sleep(8 * time.Second)

		// 找一个仍在运行的节点
		var runningIdx int
		for i, s := range servers {
			if s != nil {
				runningIdx = i
				break
			}
		}

		// 从存活的节点获取新的领导者信息
		resp, err = http.Get(baseURLs[runningIdx] + "/api/v1/cluster/leader")
		require.NoError(t, err)
		defer resp.Body.Close()

		var newLeaderInfo map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&newLeaderInfo)
		require.NoError(t, err)

		// 验证新的领导者已经产生且不是之前的领导者
		require.Contains(t, newLeaderInfo, "node_id")
		newLeaderID := newLeaderInfo["node_id"].(string)
		t.Logf("新的领导者: %s", newLeaderID)

		assert.NotEqual(t, currentLeaderID, newLeaderID, "新的领导者应该与之前的不同")
	})

	t.Run("ClusterStatusTest", func(t *testing.T) {
		// 找一个正在运行的节点
		var runningIdx int
		for i, s := range servers {
			if s != nil {
				runningIdx = i
				break
			}
		}

		// 获取集群状态
		resp, err := http.Get(baseURLs[runningIdx] + "/api/v1/cluster/status")
		require.NoError(t, err)
		defer resp.Body.Close()

		var status map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&status)
		require.NoError(t, err)

		// 验证状态包含关键指标
		assert.Contains(t, status, "node_count")
		assert.Contains(t, status, "healthy_nodes")
		assert.Contains(t, status, "leader")
		assert.Contains(t, status, "uptime")
		assert.Contains(t, status, "last_election")
		assert.Contains(t, status, "rebalance_status")
	})
}
