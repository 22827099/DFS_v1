package http_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/22827099/DFS_v1/common/network/http"
	"github.com/stretchr/testify/assert"
)

func TestClientServerIntegration(t *testing.T) {
	// 创建服务器
	server := http.NewServer(":8888")

	// 添加路由
	server.GET("/api/data", func(c *http.Context) {
		http.WriteSuccess(c.Response, map[string]string{
			"id":   c.Request.URL.Query().Get("id"),
			"time": time.Now().Format(time.RFC3339),
		})
	})

	server.POST("/api/submit", func(c *http.Context) {
		var data map[string]interface{}
		if err := json.NewDecoder(c.Request.Body).Decode(&data); err != nil {
			http.WriteError(c.Response, http.StatusBadRequest, "无效的JSON", 1001)
			return
		}
		data["processed"] = true
		http.WriteSuccess(c.Response, data)
	})

	// 启动服务器
	go func() {
		server.Start()
	}()

	// 确保服务器启动
	time.Sleep(100 * time.Millisecond)

	t.Logf("服务器启动在端口: 8888")

	// 创建客户端
	client := http.NewClient("http://localhost:8888")

	// 测试GET请求
	var getResult map[string]interface{}
	t.Logf("发送请求: GET /api/data?id=123")
	// 直接在URL中添加查询参数
	err := client.GetJSON(context.Background(), "/api/data?id=123", nil, &getResult)

	// 在访问 getResult 前检查错误和数据存在性
	if assert.NoError(t, err) {
		assert.NotNil(t, getResult["data"])
		if getResult["data"] != nil {
			dataMap := getResult["data"].(map[string]interface{})
			assert.Equal(t, "123", dataMap["id"])
		}
		t.Logf("收到响应: %+v", getResult)
	}

	// 测试POST请求
	postData := map[string]interface{}{"name": "测试", "value": 100}
	var postResult map[string]interface{}
	err = client.PostJSON(context.Background(), "/api/submit", postData, &postResult, nil)
	assert.NoError(t, err)

	resultData := postResult["data"].(map[string]interface{})
	assert.Equal(t, "测试", resultData["name"])
	assert.Equal(t, float64(100), resultData["value"])
	assert.True(t, resultData["processed"].(bool))

	// 关闭服务器
	server.Stop(context.Background())
}
