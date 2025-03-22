package metaserver_test

import (
	"testing"
	"time"

	"github.com/22827099/DFS_v1/common/config"
	"github.com/22827099/DFS_v1/internal/metaserver/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerLifecycle(t *testing.T) {
	t.Run("StartStopTest", func(t *testing.T) {
		// 创建配置
		cfg := &config.SystemConfig{
			NodeID: "test-node",
			Server: config.ServerConfig{
				Host: "localhost",
				Port: 0, // 使用随机端口避免冲突
			},
		}

		// 创建服务器
		metaServer, err := server.NewServer(cfg)
		require.NoError(t, err)

		// 启动服务器
		err = metaServer.Start()
		require.NoError(t, err)
		assert.True(t, metaServer.IsRunning())

		// 停止服务器
		err = metaServer.Stop()
		require.NoError(t, err)
		assert.False(t, metaServer.IsRunning())
	})

	t.Run("ConcurrentRequestsTest", func(t *testing.T) {
		// 创建配置
		cfg := &config.SystemConfig{
			NodeID: "test-node",
			Server: config.ServerConfig{
				Host: "localhost",
				Port: 0, // 使用随机端口避免冲突
			},
		}

		// 创建服务器
		metaServer, err := server.NewServer(cfg)
		require.NoError(t, err)

		// 启动服务器
		err = metaServer.Start()
		require.NoError(t, err)

		// 给服务器一些启动时间
		time.Sleep(100 * time.Millisecond)

		// 在这里可以添加创建并发HTTP请求的测试逻辑
		// ...

		// 停止服务器
		err = metaServer.Stop()
		require.NoError(t, err)
	})
}
