package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	customhttp "github.com/22827099/DFS_v1/common/network/http"
)

func TestClientBasic(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/success" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true, "data": {"message": "ok"}}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// 创建客户端
	client := customhttp.NewClient(server.URL)

	// 测试成功请求
	var result map[string]interface{}
	err := client.GetJSON(context.Background(), "/success", nil, &result)

	assert.NoError(t, err)
	assert.Contains(t, result, "success")
	assert.Contains(t, result, "data")
}

func TestClientRetry(t *testing.T) {
	// 计数器，记录请求次数
	requestCount := 0

	// 创建测试服务器，前两次请求返回500，第三次成功
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// 创建带重试策略的客户端
	client := customhttp.NewClient(server.URL, customhttp.WithRetryPolicy(&customhttp.RetryPolicy{
		MaxRetries:    3,
		RetryInterval: 10 * time.Millisecond,
		MaxBackoff:    100 * time.Millisecond,
		ShouldRetry: func(resp *http.Response, err error) bool {
			return err != nil || (resp != nil && resp.StatusCode >= 500)
		},
	}))

	// 发送请求
	var result map[string]interface{}
	err := client.GetJSON(context.Background(), "/", nil, &result)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, 3, requestCount, "应该尝试了3次请求")
	assert.Contains(t, result, "success")
}
