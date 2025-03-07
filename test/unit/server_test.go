package http

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	customhttp "github.com/22827099/DFS_v1/common/network/http"
)

func TestServerBasic(t *testing.T) {
	// 使用一个确定不会冲突的高端口
	testPort := "18888"
	server := customhttp.NewServer(":" + testPort)

	// 添加测试路由
	server.GET("/test", func(c *customhttp.Context) {
		customhttp.WriteSuccess(c.Response, map[string]string{"message": "测试成功"})
	})

	// 启动服务器
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			t.Logf("服务器停止: %v", err)
		}
	}()

	// 确保服务器启动
	time.Sleep(200 * time.Millisecond)

	// 发送测试请求
	resp, err := http.Get("http://localhost:" + testPort + "/test")

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 读取响应体
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	assert.Contains(t, string(body), "测试成功")

	// 关闭服务器
	server.Stop(context.Background())
}
