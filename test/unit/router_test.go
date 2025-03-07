package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	customhttp "github.com/22827099/DFS_v1/common/network/http"
)

func TestRouterBasic(t *testing.T) {
	// 创建路由器
	router := customhttp.NewRouter()

	// 注册路由
	called := false
	router.GET("/test", func(c *customhttp.Context) {
		called = true
		c.Response.WriteHeader(http.StatusOK)
	})

	// 创建测试请求和记录响应
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// 处理请求
	router.ServeHTTP(w, req)

	// 验证结果
	assert.True(t, called, "处理函数应该被调用")
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestRouterParams(t *testing.T) {
	router := customhttp.NewRouter()

	var userID string
	router.GET("/users/{id}", func(c *customhttp.Context) {
		userID = c.Params["id"]
		c.Response.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, "123", userID)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestMiddleware(t *testing.T) {
	router := customhttp.NewRouter()

	// 添加中间件
	order := make([]string, 0)
	router.Use(func(next customhttp.HandlerFunc) customhttp.HandlerFunc {
		return func(c *customhttp.Context) {
			order = append(order, "middleware1")
			next(c)
			order = append(order, "middleware1_after")
		}
	})

	router.GET("/test", func(c *customhttp.Context) {
		order = append(order, "handler")
		c.Response.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// 验证中间件执行顺序
	assert.Equal(t, []string{"middleware1", "handler", "middleware1_after"}, order)
}
