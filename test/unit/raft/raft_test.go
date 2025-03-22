package metaserver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
	"math/rand"

	"github.com/22827099/DFS_v1/common/config"
	metaconfig "github.com/22827099/DFS_v1/internal/metaserver/config"
	"github.com/22827099/DFS_v1/internal/metaserver/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var leaderURL string

// TestRaftConsensus 测试元数据服务器的Raft一致性算法
func TestRaftConsensus(t *testing.T) {

	if testing.Short() {
		t.Skip("跳过耗时的Raft一致性算法测试")
	}

	// 创建一个由3个节点组成的集群进行测试
	clusterSize := 3
	servers := make([]*server.MetadataServer, clusterSize)
	configs := make([]*config.SystemConfig, clusterSize)
	baseURLs := make([]string, clusterSize)

	// 基础端口号
	basePort := 19200

	// 准备所有节点的配置
	for i := 0; i < clusterSize; i++ {
		nodeID := fmt.Sprintf("%d", i+1)  // 替换 fmt.Sprintf("raft-node-%d", i)
		
		// 添加随机因素到选举超时
		randomElectionTimeout := 2*time.Second + time.Duration(rand.Intn(1000))*time.Millisecond
		
		configs[i] = &config.SystemConfig{
			NodeID: nodeID,
			Server: config.ServerConfig{
				Host: "localhost",
				Port: basePort + i,
			},
			Cluster: metaconfig.ClusterConfig{
				Peers: []string{"1", "2", "3"},  // 使用节点ID列表
				PeerMap: map[string]string{
					"1": fmt.Sprintf("localhost:%d", basePort),
					"2": fmt.Sprintf("localhost:%d", basePort+1),
					"3": fmt.Sprintf("localhost:%d", basePort+2),
				},
                PeerAddresses: []string{
					fmt.Sprintf("localhost:%d", basePort),
					fmt.Sprintf("localhost:%d", basePort+1),
					fmt.Sprintf("localhost:%d", basePort+2),
				},
				ElectionTimeout:  randomElectionTimeout,
				HeartbeatTimeout: 500 * time.Millisecond,
                RebalanceEvaluationInterval: 30 * time.Second, 
			},
			Consensus: config.ConsensusConfig{
				Protocol:           "raft",
				DataDir:            fmt.Sprintf("./raft-test-data-%d", i),
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
	}()

	t.Run("LeaderElectionTest", func(t *testing.T) {
        // 启动所有节点
        for i := 0; i < clusterSize; i++ {
            var err error
            servers[i], err = server.NewServer(configs[i])
            if err != nil {
                t.Fatalf("创建服务器失败: %v", err)
            }
            require.NotNil(t, servers[i], "服务器实例不应为nil")
            
            err = servers[i].Start()
            require.NoError(t, err)
            time.Sleep(500 * time.Millisecond) // 错开启动时间
        }

		// 等待集群选出领导者
		t.Log("等待集群选举领导者...")
		time.Sleep(5 * time.Second)

		// 检查是否有领导者被选举出来
		var leaderCount int
		var leaderID string

		for i := 0; i < clusterSize; i++ {
			resp, err := http.Get(baseURLs[i] + "/api/v1/cluster/leader")
			require.NoError(t, err)

			if resp.StatusCode == http.StatusOK {
				var leaderInfo map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&leaderInfo)
				require.NoError(t, err)

				// 如果该节点成功返回领导者信息，确认领导者一致
				if nodeID, ok := leaderInfo["node_id"].(string); ok && nodeID != "" {
					if leaderID == "" {
						leaderID = nodeID
						leaderURL = baseURLs[i]
					} else if leaderID != nodeID {
						t.Errorf("发现不一致的领导者: %s 和 %s", leaderID, nodeID)
					}
					leaderCount++
				}
			}
			resp.Body.Close()
		}

		assert.Equal(t, clusterSize, leaderCount, "所有节点都应该识别出相同的领导者")
		assert.NotEmpty(t, leaderID, "集群应该选出一个领导者")
		t.Logf("集群成功选出领导者: %s", leaderID)

		// 测试领导者切换
		// 首先找出领导者节点
		var leaderIdx int
		for i, cfg := range configs {
			if cfg.NodeID == leaderID {
				leaderIdx = i
				break
			}
		}

		// 停止当前领导者
		t.Logf("停止当前领导者节点: %s", configs[leaderIdx].NodeID)
		err := servers[leaderIdx].Stop()
		require.NoError(t, err)
		servers[leaderIdx] = nil

		// 等待新的选举完成
		t.Log("等待新的领导者选举...")
		time.Sleep(10 * time.Second)

		// 验证集群选举出新领导者
		var newLeaderID string
		var newLeaderFound bool

		for i := 0; i < clusterSize; i++ {
			if i == leaderIdx || servers[i] == nil {
				continue
			}

			resp, err := http.Get(baseURLs[i] + "/api/v1/cluster/leader")
			if err != nil {
				t.Logf("无法从节点 %d 获取领导者信息: %v", i, err)
				continue
			}

			if resp.StatusCode == http.StatusOK {
				var leaderInfo map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&leaderInfo)
				if err != nil {
					t.Logf("解析响应失败: %v", err)
					resp.Body.Close()
					continue
				}

				if nodeID, ok := leaderInfo["node_id"].(string); ok && nodeID != "" && nodeID != leaderID {
					newLeaderID = nodeID
					newLeaderFound = true
					t.Logf("找到新领导者: %s", newLeaderID)
				}
			}
			resp.Body.Close()
		}

		assert.True(t, newLeaderFound, "应该选举出新的领导者")
		assert.NotEqual(t, leaderID, newLeaderID, "新领导者应该与之前的领导者不同")
	})

	t.Run("LogReplicationTest", func(t *testing.T) {
		// 清理之前的服务器
		for i := 0; i < clusterSize; i++ {
			if servers[i] != nil {
				servers[i].Stop()
				servers[i] = nil
			}
		}

        // 重新启动所有节点
        for i := 0; i < clusterSize; i++ {
            var err error
            servers[i], err = server.NewServer(configs[i])
            if err != nil {
                t.Fatalf("创建服务器失败: %v", err)
            }
            require.NotNil(t, servers[i], "服务器实例不应为nil")
            
            err = servers[i].Start()
            require.NoError(t, err)
            time.Sleep(500 * time.Millisecond)
        }

		// 等待集群选举领导者
		time.Sleep(5 * time.Second)

		// 找出当前领导者
		var leaderURL string
		var leaderID string

		for i := 0; i < clusterSize; i++ {
			resp, err := http.Get(baseURLs[i] + "/api/v1/cluster/leader")
			require.NoError(t, err)

			if resp.StatusCode == http.StatusOK {
				var leaderInfo map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&leaderInfo)
				require.NoError(t, err)
				resp.Body.Close()

				if nodeID, ok := leaderInfo["node_id"].(string); ok && nodeID != "" {
					for j, cfg := range configs {
						if cfg.NodeID == nodeID {
							leaderURL = baseURLs[j]
							leaderID = nodeID
							t.Logf("当前领导者: %s, URL: %s", leaderID, leaderURL)
							break
						}
					}
					if leaderURL != "" {
						break
					}
				}
			} else {
				resp.Body.Close()
			}
		}

		require.NotEmpty(t, leaderURL, "应该找到领导者URL")

		// 创建一些测试数据，通过领导者节点写入
		testKey := "test-consensus-key"
		testValue := map[string]interface{}{
			"name":      "consensus-test-file",
			"size":      1024,
			"type":      "test-data",
			"timestamp": time.Now().Unix(),
		}

		reqBody, err := json.Marshal(testValue)
		require.NoError(t, err)

		// 向领导者提交数据
		resp, err := http.Post(
			leaderURL+"/api/v1/kv/"+testKey,
			"application/json",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "应该成功写入数据")

		// 等待数据复制到其他节点
		time.Sleep(2 * time.Second)

		// 验证每个节点都能读取到相同的数据
		for i := 0; i < clusterSize; i++ {
			resp, err := http.Get(baseURLs[i] + "/api/v1/kv/" + testKey)
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, resp.StatusCode, "节点 %d 应该能够读取数据", i)

			if resp.StatusCode == http.StatusOK {
				var readValue map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&readValue)
				require.NoError(t, err)

				assert.Equal(t, testValue["name"], readValue["name"], "节点 %d 读取的数据不一致", i)
				assert.Equal(t, testValue["size"], readValue["size"], "节点 %d 读取的数据不一致", i)
				assert.Equal(t, testValue["type"], readValue["type"], "节点 %d 读取的数据不一致", i)
			}
			resp.Body.Close()
		}
	})

	t.Run("FaultToleranceTest", func(t *testing.T) {
		// 清理之前的服务器
		for i := 0; i < clusterSize; i++ {
			if servers[i] != nil {
				servers[i].Stop()
				servers[i] = nil
			}
		}

		// 重新启动所有节点
		for i := 0; i < clusterSize; i++ {
			var err error
            servers[i], err = server.NewServer(configs[i])
            if err != nil {
                t.Fatalf("创建服务器失败: %v", err)
            }
            require.NotNil(t, servers[i], "服务器实例不应为nil")

            err = servers[i].Start()
            require.NoError(t, err)
			time.Sleep(500 * time.Millisecond)
		}

		// 等待集群选举领导者
		time.Sleep(5 * time.Second)

		// 找出当前领导者
		var leaderURL string
		var leaderID string
		var leaderIdx int

		for i := 0; i < clusterSize; i++ {
			resp, err := http.Get(baseURLs[i] + "/api/v1/cluster/leader")
			require.NoError(t, err)

			if resp.StatusCode == http.StatusOK {
				var leaderInfo map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&leaderInfo)
				require.NoError(t, err)
				resp.Body.Close()

				if nodeID, ok := leaderInfo["node_id"].(string); ok && nodeID != "" {
					for j, cfg := range configs {
						if cfg.NodeID == nodeID {
							leaderURL = baseURLs[j]
							leaderID = nodeID
							leaderIdx = j
							t.Logf("当前领导者: %s, URL: %s", leaderID, leaderURL)
							break
						}
					}
					if leaderURL != "" {
						break
					}
				}
			} else {
				resp.Body.Close()
			}
		}

		require.NotEmpty(t, leaderURL, "应该找到领导者URL")

		// 确定一个跟随者节点作为故障节点
		var followerIdx int
		for i := 0; i < clusterSize; i++ {
			if i != leaderIdx {
				followerIdx = i
				break
			}
		}

		// 停止这个跟随者节点
		t.Logf("停止跟随者节点: %s", configs[followerIdx].NodeID)
		err := servers[followerIdx].Stop()
		require.NoError(t, err)
		servers[followerIdx] = nil

		// 在只有2个节点的情况下继续写入数据
		testKey := "fault-tolerance-key"
		testValue := map[string]interface{}{
			"name":      "fault-tolerance-test",
			"size":      2048,
			"type":      "test-data-fault",
			"timestamp": time.Now().Unix(),
		}

		reqBody, err := json.Marshal(testValue)
		require.NoError(t, err)

		// 向领导者提交数据
		resp, err := http.Post(
			leaderURL+"/api/v1/kv/"+testKey,
			"application/json",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "仍应该能够写入数据")

		// 恢复故障节点
		t.Logf("重启故障节点: %s", configs[followerIdx].NodeID)
		servers[followerIdx], _ = server.NewServer(configs[followerIdx])
		err = servers[followerIdx].Start()
		require.NoError(t, err)

		// 等待节点恢复并同步
		time.Sleep(5 * time.Second)

		// 检查恢复的节点是否同步了故障期间的数据
		resp, err = http.Get(baseURLs[followerIdx] + "/api/v1/kv/" + testKey)
		require.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var syncedValue map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&syncedValue)
			require.NoError(t, err)

			assert.Equal(t, testValue["name"], syncedValue["name"], "恢复的节点应该同步故障期间写入的数据")
			assert.Equal(t, testValue["size"], syncedValue["size"], "恢复的节点应该同步故障期间写入的数据")
			assert.Equal(t, testValue["type"], syncedValue["type"], "恢复的节点应该同步故障期间写入的数据")

			t.Log("恢复的节点成功同步了故障期间的数据")
		} else {
			t.Errorf("恢复的节点未能读取到故障期间写入的数据，状态码: %d", resp.StatusCode)
		}
	})

	t.Run("ConfigurationChangeTest", func(t *testing.T) {
		// 清理之前的服务器
		for i := 0; i < clusterSize; i++ {
			if servers[i] != nil {
				servers[i].Stop()
				servers[i] = nil
			}
		}

		// 创建一个初始只有2个节点的集群
		initialClusterSize := 2
		for i := 0; i < initialClusterSize; i++ {
			// 修改配置，初始只包含2个节点
            configs[i].Cluster.Peers = []string{"1", "2"}

			var err error
            servers[i], err = server.NewServer(configs[i])
            if err != nil {
                t.Fatalf("创建服务器失败: %v", err)
            }
            require.NotNil(t, servers[i], "服务器实例不应为nil")
			err = servers[i].Start()
			require.NoError(t, err)
			time.Sleep(500 * time.Millisecond)
		}

		// 等待集群选举领导者
		time.Sleep(5 * time.Second)

		// 找出当前领导者
		var leaderURL string

		for i := 0; i < initialClusterSize; i++ {
			resp, err := http.Get(baseURLs[i] + "/api/v1/cluster/leader")
			require.NoError(t, err)

			if resp.StatusCode == http.StatusOK {
				var leaderInfo map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&leaderInfo)
				require.NoError(t, err)
				resp.Body.Close()

				if nodeID, ok := leaderInfo["node_id"].(string); ok && nodeID != "" {
					for j := 0; j < initialClusterSize; j++ {
						if configs[j].NodeID == nodeID {
							leaderURL = baseURLs[j]
							t.Logf("当前领导者: %s, URL: %s", nodeID, leaderURL)
							break
						}
					}
					if leaderURL != "" {
						break
					}
				}
			} else {
				resp.Body.Close()
			}
		}

		require.NotEmpty(t, leaderURL, "应该找到领导者URL")

		// 准备添加第三个节点
		newNodeConfig := map[string]interface{}{
			"node_id": configs[2].NodeID,
			"address": fmt.Sprintf("localhost:%d", basePort+2),
		}

		reqBody, err := json.Marshal(newNodeConfig)
		require.NoError(t, err)

		// 通过API添加新节点到集群
		t.Logf("添加新节点: %s 到集群", configs[2].NodeID)
		resp, err := http.Post(
			leaderURL+"/api/v1/cluster/nodes",
			"application/json",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "应该成功添加新节点")

		// 启动新的节点
		configs[2].Cluster.Peers = []string{"1", "2", "3"}

		servers[2], _ = server.NewServer(configs[2])
		err = servers[2].Start()
		require.NoError(t, err)

		// 等待节点加入集群
		time.Sleep(5 * time.Second)

		// 检查集群节点状态
		resp, err = http.Get(leaderURL + "/api/v1/cluster/nodes")
		require.NoError(t, err)
		defer resp.Body.Close()

		var clusterNodes map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&clusterNodes)
		require.NoError(t, err)

		nodesList, ok := clusterNodes["nodes"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 3, len(nodesList), "集群应该有3个节点")

		// 向集群写入一些数据，验证新节点是否参与一致性协议
		testKey := "config-change-key"
		testValue := map[string]interface{}{
			"name":      "config-change-test",
			"size":      4096,
			"type":      "test-data-config",
			"timestamp": time.Now().Unix(),
		}

		reqBody, err = json.Marshal(testValue)
		require.NoError(t, err)

		// 向领导者提交数据
		resp, err = http.Post(
			leaderURL+"/api/v1/kv/"+testKey,
			"application/json",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err)
		resp.Body.Close()

		// 验证所有节点（包括新加入的）都能读取数据
		time.Sleep(2 * time.Second)

		resp, err = http.Get(baseURLs[2] + "/api/v1/kv/" + testKey)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "新节点应该能够读取数据")

		if resp.StatusCode == http.StatusOK {
			var readValue map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&readValue)
			require.NoError(t, err)

			assert.Equal(t, testValue["name"], readValue["name"], "新节点读取的数据应一致")
			assert.Equal(t, testValue["size"], readValue["size"], "新节点读取的数据应一致")
			assert.Equal(t, testValue["type"], readValue["type"], "新节点读取的数据应一致")

			t.Log("新节点成功参与了一致性协议")
		}
	})
}
