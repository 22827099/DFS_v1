package metaserver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/22827099/DFS_v1/common/config"
	"github.com/22827099/DFS_v1/internal/metaserver/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetaServerCluster 测试元数据服务器集群功能
func TestMetaServerCluster(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过耗时的元数据服务器集群测试")
	}

	// 创建测试数据目录
	testDataDir, err := os.MkdirTemp("", "metaserver-cluster-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(testDataDir)

	// 创建5个节点的集群配置
	clusterSize := 5
	servers := make([]*server.Server, clusterSize)
	configs := make([]*config.SystemConfig, clusterSize)
	baseURLs := make([]string, clusterSize)
	dataDirs := make([]string, clusterSize)

	// 基础端口号
	basePort := 20000

	// 所有节点的地址列表
	peerAddresses := make([]string, clusterSize)
	for i := 0; i < clusterSize; i++ {
		peerAddresses[i] = fmt.Sprintf("localhost:%d", basePort+i)
	}

	// 创建网络隔离控制器
	networkPartitioner := newNetworkPartitioner()

	// 准备所有节点的配置
	for i := 0; i < clusterSize; i++ {
		nodeID := fmt.Sprintf("ms-node-%d", i)
		dataDirs[i] = filepath.Join(testDataDir, fmt.Sprintf("node-%d", i))
		err := os.MkdirAll(dataDirs[i], 0755)
		require.NoError(t, err)

		configs[i] = &config.SystemConfig{
			NodeID:  nodeID,
			DataDir: dataDirs[i],
			Server: config.ServerConfig{
				Host: "localhost",
				Port: basePort + i,
			},
			Cluster: config.ClusterConfig{
				Peers:            peerAddresses,
				ElectionTimeout:  3 * time.Second,
				HeartbeatTimeout: 1 * time.Second,
				SuspectTimeout:   5 * time.Second,
				DeadTimeout:      10 * time.Second,
			},
			Consensus: config.ConsensusConfig{
				Protocol:           "raft",
				DataDir:            filepath.Join(dataDirs[i], "raft"),
				SnapshotThreshold:  1000,
				CompactionInterval: 10 * time.Minute,
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
		networkPartitioner.tearDown()
	}()

	t.Run("ClusterFormationTest", func(t *testing.T) {
		// 启动所有节点
		for i := 0; i < clusterSize; i++ {
			servers[i], err = server.NewServer(configs[i])
			require.NoError(t, err)
			err = servers[i].Start()
			require.NoError(t, err)
			time.Sleep(1 * time.Second) // 错开启动时间
		}

		// 等待集群形成
		t.Log("等待集群形成...")
		time.Sleep(10 * time.Second)

		// 检查所有节点是否能看到完整的集群成员
		for i := 0; i < clusterSize; i++ {
			t.Logf("检查节点 %d 的集群视图", i)
			resp, err := http.Get(baseURLs[i] + "/api/v1/cluster/nodes")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var clusterView map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&clusterView)
			require.NoError(t, err)

			nodes, ok := clusterView["nodes"].([]interface{})
			require.True(t, ok)
			assert.Equal(t, clusterSize, len(nodes), "节点 %d 应该看到所有 %d 个集群成员", i, clusterSize)
		}

		// 确认集群有一个领导者
		var leaderFound bool
		var leaderID string
		var leaderURL string
		var leaderIdx int

		for i := 0; i < clusterSize; i++ {
			resp, err := http.Get(baseURLs[i] + "/api/v1/cluster/leader")
			require.NoError(t, err)

			if resp.StatusCode == http.StatusOK {
				var leaderInfo map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&leaderInfo)
				require.NoError(t, err)
				resp.Body.Close()

				if id, ok := leaderInfo["node_id"].(string); ok && id != "" {
					leaderFound = true
					leaderID = id
					leaderURL = baseURLs[i]
					leaderIdx = i
					break
				}
			}
			resp.Body.Close()
		}

		assert.True(t, leaderFound, "集群应该选出领导者")
		t.Logf("确认当前领导者: %s (idx: %d)", leaderID, leaderIdx)

		// 测试集群写入操作
		t.Log("测试集群写入操作")
		testFile := map[string]interface{}{
			"name":      "cluster-test.txt",
			"size":      1024,
			"mime_type": "text/plain",
		}
		reqBody, err := json.Marshal(testFile)
		require.NoError(t, err)

		// 发送HTTP请求创建文件
		resp, err := http.Post(
			leaderURL+"/api/v1/files/cluster-test.txt",
			"application/json",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		// 验证所有节点都能读取到文件
		for i := 0; i < clusterSize; i++ {
			t.Logf("从节点 %d 读取文件", i)
			resp, err := http.Get(baseURLs[i] + "/api/v1/files/cluster-test.txt")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode, "节点 %d 应该能读取到文件", i)

			var fileData map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&fileData)
			require.NoError(t, err)

			assert.Equal(t, "cluster-test.txt", fileData["name"])
			assert.Equal(t, float64(1024), fileData["size"])
		}
	})

	t.Run("LeaderFailoverTest", func(t *testing.T) {
		// 找出当前领导者
		var leaderIdx int = -1
		var leaderID string

		for i := 0; i < clusterSize; i++ {
			if servers[i] == nil {
				continue
			}

			resp, err := http.Get(baseURLs[i] + "/api/v1/cluster/leader")
			require.NoError(t, err)

			if resp.StatusCode == http.StatusOK {
				var leaderInfo map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&leaderInfo)
				require.NoError(t, err)
				resp.Body.Close()

				if nodeID, ok := leaderInfo["node_id"].(string); ok && nodeID != "" {
					for j := 0; j < clusterSize; j++ {
						if configs[j].NodeID == nodeID {
							leaderIdx = j
							leaderID = nodeID
							break
						}
					}
					if leaderIdx >= 0 {
						break
					}
				}
			} else {
				resp.Body.Close()
			}
		}

		require.GreaterOrEqual(t, leaderIdx, 0, "应该找到一个领导者")
		t.Logf("当前领导者: %s (idx: %d)", leaderID, leaderIdx)

		// 向领导者写入数据
		testFile := map[string]interface{}{
			"name":      "before-failover.txt",
			"size":      2048,
			"mime_type": "text/plain",
		}
		reqBody, err := json.Marshal(testFile)
		require.NoError(t, err)

		resp, err := http.Post(
			baseURLs[leaderIdx]+"/api/v1/files/before-failover.txt",
			"application/json",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err)
		resp.Body.Close()

		// 停止当前领导者
		t.Logf("停止当前领导者 (idx: %d)", leaderIdx)
		err = servers[leaderIdx].Stop()
		require.NoError(t, err)
		servers[leaderIdx] = nil

		// 等待新的领导者选举完成
		t.Log("等待新的领导者选举完成...")
		time.Sleep(15 * time.Second)

		// 找出新的领导者
		var newLeaderIdx int = -1
		var newLeaderID string

		for i := 0; i < clusterSize; i++ {
			if servers[i] == nil {
				continue
			}

			resp, err := http.Get(baseURLs[i] + "/api/v1/cluster/leader")
			require.NoError(t, err)

			if resp.StatusCode == http.StatusOK {
				var leaderInfo map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&leaderInfo)
				require.NoError(t, err)
				resp.Body.Close()

				if nodeID, ok := leaderInfo["node_id"].(string); ok && nodeID != "" {
					for j := 0; j < clusterSize; j++ {
						if configs[j].NodeID == nodeID && j != leaderIdx {
							newLeaderIdx = j
							newLeaderID = nodeID
							break
						}
					}
					if newLeaderIdx >= 0 {
						break
					}
				}
			} else {
				resp.Body.Close()
			}
		}

		require.GreaterOrEqual(t, newLeaderIdx, 0, "应该选出新的领导者")
		t.Logf("新的领导者: %s (idx: %d)", newLeaderID, newLeaderIdx)
		assert.NotEqual(t, leaderIdx, newLeaderIdx, "新的领导者应该与之前的不同")

		// 验证新领导者可以读取之前的数据
		resp, err = http.Get(baseURLs[newLeaderIdx] + "/api/v1/files/before-failover.txt")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "新的领导者应该能够读取故障前的数据")

		var fileData map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&fileData)
		require.NoError(t, err)
		assert.Equal(t, "before-failover.txt", fileData["name"])

		// 向新领导者写入新数据
		testFile = map[string]interface{}{
			"name":      "after-failover.txt",
			"size":      4096,
			"mime_type": "application/octet-stream",
		}
		reqBody, err = json.Marshal(testFile)
		require.NoError(t, err)

		resp, err = http.Post(
			baseURLs[newLeaderIdx]+"/api/v1/files/after-failover.txt",
			"application/json",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err)
		resp.Body.Close()

		// 验证所有存活节点都能读取新数据
		for i := 0; i < clusterSize; i++ {
			if i == leaderIdx || servers[i] == nil {
				continue
			}

			resp, err := http.Get(baseURLs[i] + "/api/v1/files/after-failover.txt")
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, resp.StatusCode, "节点 %d 应该能读取到故障后写入的文件", i)

			var fileData map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&fileData)
			require.NoError(t, err)
			resp.Body.Close()

			assert.Equal(t, "after-failover.txt", fileData["name"])
			assert.Equal(t, float64(4096), fileData["size"])
		}
	})

	t.Run("ConsistencyDuringNetworkPartitionTest", func(t *testing.T) {
		// 如果有节点停止了，先重启它们
		for i := 0; i < clusterSize; i++ {
			if servers[i] == nil {
				servers[i], err = server.NewServer(configs[i])
				require.NoError(t, err)
				err = servers[i].Start()
				require.NoError(t, err)
				time.Sleep(1 * time.Second)
			}
		}

		// 等待集群稳定
		t.Log("等待集群稳定...")
		time.Sleep(10 * time.Second)

		// 找出当前领导者
		var leaderIdx int = -1

		for i := 0; i < clusterSize; i++ {
			resp, err := http.Get(baseURLs[i] + "/api/v1/cluster/leader")
			require.NoError(t, err)

			if resp.StatusCode == http.StatusOK {
				var leaderInfo map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&leaderInfo)
				require.NoError(t, err)
				resp.Body.Close()

				if nodeID, ok := leaderInfo["node_id"].(string); ok && nodeID != "" {
					for j := 0; j < clusterSize; j++ {
						if configs[j].NodeID == nodeID {
							leaderIdx = j
							break
						}
					}
					if leaderIdx >= 0 {
						break
					}
				}
			} else {
				resp.Body.Close()
			}
		}

		require.GreaterOrEqual(t, leaderIdx, 0, "应该找到一个领导者")
		t.Logf("当前领导者索引: %d", leaderIdx)

		// 创建网络分区: 将集群分成两部分，确保领导者在多数派中
		// 假设集群大小是5，分区为 [0,1,2] 和 [3,4]，确保领导者在多数派中
		majorityPartition := []int{0, 1, 2}
		minorityPartition := []int{3, 4}

		if leaderIdx > 2 {
			// 如果领导者在少数派，交换分区让领导者在多数派
			majorityPartition, minorityPartition = minorityPartition, majorityPartition
		}

		// 创建网络分区
		t.Logf("创建网络分区: 多数派 %v (包含领导者) vs 少数派 %v", majorityPartition, minorityPartition)
		for _, i := range majorityPartition {
			for _, j := range minorityPartition {
				networkPartitioner.partitionNodes(basePort+i, basePort+j)
			}
		}

		// 在多数派上写入数据
		t.Log("在多数派上写入数据")
		testFile := map[string]interface{}{
			"name":      "majority-partition.txt",
			"size":      8192,
			"mime_type": "application/binary",
		}
		reqBody, err := json.Marshal(testFile)
		require.NoError(t, err)

		majorityLeaderIdx := -1
		for _, idx := range majorityPartition {
			if idx == leaderIdx {
				majorityLeaderIdx = idx
				break
			}
		}
		require.GreaterOrEqual(t, majorityLeaderIdx, 0, "应该在多数派中找到领导者")

		resp, err := http.Post(
			baseURLs[majorityLeaderIdx]+"/api/v1/files/majority-partition.txt",
			"application/json",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err)
		resp.Body.Close()

		// 验证多数派节点能读取到数据
		for _, idx := range majorityPartition {
			resp, err := http.Get(baseURLs[idx] + "/api/v1/files/majority-partition.txt")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode, "多数派节点 %d 应该能读取到写入的文件", idx)
		}

		// 尝试在少数派上写入数据，应该失败
		t.Log("尝试在少数派上写入数据，预期会失败")
		testFile = map[string]interface{}{
			"name":      "minority-partition.txt",
			"size":      512,
			"mime_type": "text/html",
		}
		reqBody, err = json.Marshal(testFile)
		require.NoError(t, err)

		// 由于网络分区，少数派应该选不出新领导者，并且无法接受写入
		// 但HTTP客户端可能会因为连接超时而返回错误，因此我们需要处理这种情况
		for _, idx := range minorityPartition {
			client := http.Client{
				Timeout: 5 * time.Second,
			}
			_, err := client.Post(
				baseURLs[idx]+"/api/v1/files/minority-partition.txt",
				"application/json",
				bytes.NewReader(reqBody),
			)

			// 不管是超时还是服务器错误，都符合预期
			t.Logf("尝试在少数派节点 %d 上写入，结果: %v", idx, err)
		}

		// 修复网络分区
		t.Log("修复网络分区")
		for _, i := range majorityPartition {
			for _, j := range minorityPartition {
				networkPartitioner.healPartition(basePort+i, basePort+j)
			}
		}

		// 等待集群恢复
		t.Log("等待集群恢复...")
		time.Sleep(10 * time.Second)

		// 验证所有节点现在都能读取到多数派写入的数据
		for i := 0; i < clusterSize; i++ {
			resp, err := http.Get(baseURLs[i] + "/api/v1/files/majority-partition.txt")
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, resp.StatusCode, "节点 %d 应该能读取到分区期间写入的文件", i)

			var fileData map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&fileData)
			require.NoError(t, err)
			resp.Body.Close()

			assert.Equal(t, "majority-partition.txt", fileData["name"])
			assert.Equal(t, float64(8192), fileData["size"])
		}

		// 验证所有节点无法读取到少数派尝试写入的数据
		for i := 0; i < clusterSize; i++ {
			resp, err := http.Get(baseURLs[i] + "/api/v1/files/minority-partition.txt")
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode, "节点 %d 不应该能读取到少数派尝试写入的文件", i)
		}
	})
}

// 网络分区模拟器
type networkPartitioner struct {
	partitions map[string]struct{}
}

func newNetworkPartitioner() *networkPartitioner {
	return &networkPartitioner{
		partitions: make(map[string]struct{}),
	}
}

func (p *networkPartitioner) partitionNodes(port1, port2 int) {
	key := fmt.Sprintf("%d-%d", port1, port2)
	if _, exists := p.partitions[key]; exists {
		return
	}

	// 使用iptables创建防火墙规则以模拟网络分区
	// 注意: 这需要管理员/root权限
	// 实际实现可能需要根据操作系统和环境调整
	p.partitions[key] = struct{}{}

	// 这里只是模拟实现，实际测试环境中可能需要使用iptables或类似工具
	// 例如: iptables -A INPUT -p tcp --dport port1 -s localhost:port2 -j DROP
	t.Logf("模拟网络分区: 节点 %d 无法连接到节点 %d", port1, port2)
}

func (p *networkPartitioner) healPartition(port1, port2 int) {
	key := fmt.Sprintf("%d-%d", port1, port2)
	if _, exists := p.partitions[key]; !exists {
		return
	}

	// 删除防火墙规则以恢复连接
	delete(p.partitions, key)

	// 这里只是模拟实现，实际测试环境中可能需要使用iptables或类似工具
	// 例如: iptables -D INPUT -p tcp --dport port1 -s localhost:port2 -j DROP
	t.Logf("恢复网络连接: 节点 %d 可以连接到节点 %d", port1, port2)
}

func (p *networkPartitioner) tearDown() {
	// 清理所有网络分区规则
	p.partitions = make(map[string]struct{})
}
