package metaserver_test

import (
	"bytes"
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

// TestLoadBalancer 测试负载均衡功能
func TestLoadBalancer(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过耗时的负载均衡测试")
	}

	// 创建3个节点的集群配置
	clusterSize := 3
	servers := make([]*server.Server, clusterSize)
	configs := make([]*config.SystemConfig, clusterSize)
	baseURLs := make([]string, clusterSize)

	// 基础端口号
	basePort := 19100

	// 准备所有节点的配置
	for i := 0; i < clusterSize; i++ {
		nodeID := fmt.Sprintf("lb-node-%d", i)
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
			LoadBalancer: config.LoadBalancerConfig{
				EvaluationInterval:      30 * time.Second,
				ImbalanceThreshold:      20.0,
				MaxConcurrentMigrations: 2,
				MinMigrationInterval:    1 * time.Minute,
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

	// 启动集群节点
	for i := 0; i < clusterSize; i++ {
		servers[i], _ = server.NewServer(configs[i])
		err := servers[i].Start()
		require.NoError(t, err)
		time.Sleep(500 * time.Millisecond) // 错开启动时间
	}

	// 等待集群稳定
	time.Sleep(5 * time.Second)

	// 确定领导者节点
	var leaderURL string
	for i := 0; i < clusterSize; i++ {
		resp, err := http.Get(baseURLs[i] + "/api/v1/cluster/leader")
		require.NoError(t, err)
		if resp.StatusCode == http.StatusOK {
			var leaderInfo map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&leaderInfo)
			require.NoError(t, err)
			resp.Body.Close()

			for j := 0; j < clusterSize; j++ {
				if configs[j].NodeID == leaderInfo["node_id"].(string) {
					leaderURL = baseURLs[j]
					t.Logf("领导者节点: %s", configs[j].NodeID)
					break
				}
			}
			break
		}
		resp.Body.Close()
	}
	require.NotEmpty(t, leaderURL, "无法确定领导者节点")

	t.Run("MetricsCollectionTest", func(t *testing.T) {
		// 为每个节点提交不同的指标数据
		for i, cfg := range configs {
			metrics := map[string]interface{}{
				"cpu_usage":     float64(30 + i*20),
				"memory_usage":  float64(40 + i*15),
				"disk_usage":    float64(50 + i*10),
				"total_storage": uint64(1024 * 1024 * 1024 * (100 - i*20)),
				"used_storage":  uint64(1024 * 1024 * 1024 * (30 + i*15)),
				"shard_count":   uint32(100 + i*50),
				"timestamp":     time.Now().Unix(),
			}

			reqBody, err := json.Marshal(metrics)
			require.NoError(t, err)

			// 发送指标到领导者节点
			resp, err := http.Post(
				leaderURL+"/api/v1/cluster/metrics/"+cfg.NodeID,
				"application/json",
				bytes.NewReader(reqBody),
			)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}

		// 验证集群整体指标
		resp, err := http.Get(leaderURL + "/api/v1/cluster/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()

		var clusterMetrics map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&clusterMetrics)
		require.NoError(t, err)

		// 验证所有节点的指标都已收集
		nodesMetrics, ok := clusterMetrics["nodes"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, clusterSize, len(nodesMetrics))

		// 验证集群统计信息
		stats, ok := clusterMetrics["stats"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, stats, "node_count")
		assert.Contains(t, stats, "total_storage")
		assert.Contains(t, stats, "used_storage")
		assert.Contains(t, stats, "avg_cpu_usage")
		assert.Contains(t, stats, "avg_memory_usage")
		assert.Contains(t, stats, "avg_disk_usage")
	})

	t.Run("ImbalanceDetectionTest", func(t *testing.T) {
		// 创建明显不平衡的节点指标
		unbalancedMetrics := []map[string]interface{}{
			{
				"cpu_usage":     float64(90),
				"memory_usage":  float64(80),
				"disk_usage":    float64(85),
				"total_storage": uint64(1024 * 1024 * 1024 * 100),
				"used_storage":  uint64(1024 * 1024 * 1024 * 80),
				"shard_count":   uint32(500),
				"timestamp":     time.Now().Unix(),
			},
			{
				"cpu_usage":     float64(30),
				"memory_usage":  float64(20),
				"disk_usage":    float64(25),
				"total_storage": uint64(1024 * 1024 * 1024 * 100),
				"used_storage":  uint64(1024 * 1024 * 1024 * 20),
				"shard_count":   uint32(100),
				"timestamp":     time.Now().Unix(),
			},
			{
				"cpu_usage":     float64(20),
				"memory_usage":  float64(15),
				"disk_usage":    float64(20),
				"total_storage": uint64(1024 * 1024 * 1024 * 100),
				"used_storage":  uint64(1024 * 1024 * 1024 * 15),
				"shard_count":   uint32(80),
				"timestamp":     time.Now().Unix(),
			},
		}

		// 提交不平衡的指标
		for i, cfg := range configs {
			reqBody, err := json.Marshal(unbalancedMetrics[i])
			require.NoError(t, err)

			resp, err := http.Post(
				leaderURL+"/api/v1/cluster/metrics/"+cfg.NodeID,
				"application/json",
				bytes.NewReader(reqBody),
			)
			require.NoError(t, err)
			resp.Body.Close()
		}

		// 获取不平衡状态评估
		resp, err := http.Get(leaderURL + "/api/v1/cluster/balance/status")
		require.NoError(t, err)
		defer resp.Body.Close()

		var balanceStatus map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&balanceStatus)
		require.NoError(t, err)

		// 验证检测到了不平衡状态
		assert.Contains(t, balanceStatus, "is_balanced")
		assert.Contains(t, balanceStatus, "imbalance_score")
		assert.Contains(t, balanceStatus, "threshold")

		isBalanced, ok := balanceStatus["is_balanced"].(bool)
		require.True(t, ok)
		assert.False(t, isBalanced, "应检测到不平衡状态")

		imbalanceScore, ok := balanceStatus["imbalance_score"].(float64)
		require.True(t, ok)
		threshold, ok := balanceStatus["threshold"].(float64)
		require.True(t, ok)

		assert.Greater(t, imbalanceScore, threshold, "不平衡得分应超过阈值")
	})

	t.Run("MigrationPlanGenerationTest", func(t *testing.T) {
		// 触发迁移计划生成
		resp, err := http.Post(leaderURL+"/api/v1/cluster/balance/plan", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		var planResult map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&planResult)
		require.NoError(t, err)

		// 验证生成的迁移计划
		assert.Contains(t, planResult, "plans")
		plans, ok := planResult["plans"].([]interface{})
		require.True(t, ok)

		// 应该至少有一个迁移计划
		assert.NotEmpty(t, plans, "应生成至少一个迁移计划")

		// 验证计划内容
		plan := plans[0].(map[string]interface{})
		assert.Contains(t, plan, "source_node_id")
		assert.Contains(t, plan, "target_node_id")
		assert.Contains(t, plan, "shard_ids")
		assert.Contains(t, plan, "estimated_bytes")

		// 源节点应该是负载最重的节点
		assert.Equal(t, configs[0].NodeID, plan["source_node_id"])
	})

	t.Run("TaskExecutionTest", func(t *testing.T) {
		// 为了测试任务执行，我们直接提交一个迁移任务
		migrationTask := map[string]interface{}{
			"source_node_id":  configs[0].NodeID,
			"target_node_id":  configs[2].NodeID,
			"shard_ids":       []string{"shard-1", "shard-2", "shard-3"},
			"estimated_bytes": uint64(1024 * 1024 * 10),
			"priority":        10,
		}

		reqBody, err := json.Marshal(migrationTask)
		require.NoError(t, err)

		// 提交任务
		resp, err := http.Post(
			leaderURL+"/api/v1/cluster/balance/tasks",
			"application/json",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		var taskResult map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&taskResult)
		require.NoError(t, err)

		// 验证任务创建成功
		assert.Contains(t, taskResult, "task_id")
		taskID := taskResult["task_id"].(string)
		assert.NotEmpty(t, taskID)

		// 等待任务执行
		time.Sleep(3 * time.Second)

		// 获取任务状态
		resp, err = http.Get(leaderURL + "/api/v1/cluster/balance/tasks/" + taskID)
		require.NoError(t, err)
		defer resp.Body.Close()

		var taskStatus map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&taskStatus)
		require.NoError(t, err)

		// 验证任务状态
		assert.Contains(t, taskStatus, "status")
		assert.Contains(t, []string{"completed", "running"}, taskStatus["status"].(string))
	})

	t.Run("TaskRetryTest", func(t *testing.T) {
		// 创建一个会失败的任务（使用不存在的分片ID）
		failingTask := map[string]interface{}{
			"source_node_id":  configs[0].NodeID,
			"target_node_id":  configs[1].NodeID,
			"shard_ids":       []string{"non-existent-shard"},
			"estimated_bytes": uint64(1024 * 1024),
			"priority":        8,
			"should_fail":     true, // 特殊标记，使服务器模拟失败
		}

		reqBody, err := json.Marshal(failingTask)
		require.NoError(t, err)

		// 提交任务
		resp, err := http.Post(
			leaderURL+"/api/v1/cluster/balance/tasks",
			"application/json",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		var taskResult map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&taskResult)
		require.NoError(t, err)

		taskID := taskResult["task_id"].(string)

		// 等待任务执行几次重试
		time.Sleep(8 * time.Second)

		// 获取任务状态
		resp, err = http.Get(leaderURL + "/api/v1/cluster/balance/tasks/" + taskID)
		require.NoError(t, err)
		defer resp.Body.Close()

		var taskStatus map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&taskStatus)
		require.NoError(t, err)

		// 验证任务已重试
		assert.Contains(t, taskStatus, "retry_count")
		retryCount, ok := taskStatus["retry_count"].(float64)
		require.True(t, ok)
		assert.Greater(t, retryCount, float64(0), "任务应该已重试")
	})

	t.Run("ConcurrencyLimitTest", func(t *testing.T) {
		// 创建多个任务，测试并发限制
		for i := 0; i < 5; i++ {
			task := map[string]interface{}{
				"source_node_id":  configs[0].NodeID,
				"target_node_id":  configs[2].NodeID,
				"shard_ids":       []string{fmt.Sprintf("concurrent-shard-%d", i)},
				"estimated_bytes": uint64(1024 * 1024 * 5),
				"priority":        5 - i,
			}

			reqBody, err := json.Marshal(task)
			require.NoError(t, err)

			resp, err := http.Post(
				leaderURL+"/api/v1/cluster/balance/tasks",
				"application/json",
				bytes.NewReader(reqBody),
			)
			require.NoError(t, err)
			resp.Body.Close()
		}

		// 等待任务调度
		time.Sleep(2 * time.Second)

		// 获取活跃任务状态
		resp, err := http.Get(leaderURL + "/api/v1/cluster/balance/tasks/active")
		require.NoError(t, err)
		defer resp.Body.Close()

		var activeTasksResult map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&activeTasksResult)
		require.NoError(t, err)

		// 验证活跃任务数不超过配置的限制
		assert.Contains(t, activeTasksResult, "active_tasks")
		activeTasks, ok := activeTasksResult["active_tasks"].([]interface{})
		require.True(t, ok)

		assert.LessOrEqual(t, len(activeTasks), configs[0].LoadBalancer.MaxConcurrentMigrations,
			"活跃任务数应不超过最大并发限制")

		// 获取挂起的任务
		resp, err = http.Get(leaderURL + "/api/v1/cluster/balance/tasks/pending")
		require.NoError(t, err)
		defer resp.Body.Close()

		var pendingTasksResult map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&pendingTasksResult)
		require.NoError(t, err)

		// 验证有任务在等待
		assert.Contains(t, pendingTasksResult, "pending_tasks")
		pendingTasks, ok := pendingTasksResult["pending_tasks"].([]interface{})
		require.True(t, ok)

		// 应该有任务在等待（总提交5个，最大并发2个）
		assert.NotEmpty(t, pendingTasks, "应该有任务在等待执行")
	})

	t.Run("TriggerManualRebalanceTest", func(t *testing.T) {
		// 手动触发负载均衡
		resp, err := http.Post(leaderURL+"/api/v1/cluster/balance/trigger", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 验证负载均衡状态
		resp, err = http.Get(leaderURL + "/api/v1/cluster/balance/status")
		require.NoError(t, err)
		defer resp.Body.Close()

		var rebalanceStatus map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&rebalanceStatus)
		require.NoError(t, err)

		assert.Contains(t, rebalanceStatus, "is_rebalancing")
	})
}
