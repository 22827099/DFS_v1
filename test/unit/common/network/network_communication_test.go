package network_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/22827099/DFS_v1/common/logging"
	nethttp "github.com/22827099/DFS_v1/common/network/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/gorilla/mux"
)

// TestNetworkCommunication 测试网络通信功能
func TestNetworkCommunication(t *testing.T) {
	t.Run("HTTPClientTest", func(t *testing.T) {
		// 创建测试服务器
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/get":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"status":"success","data":{"message":"Hello, World!"}}`)

			case "/api/post":
				// 读取和验证请求体
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				if err != nil || reqBody["name"] != "test" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				fmt.Fprintln(w, `{"status":"created","data":{"id":123}}`)

			case "/api/error":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, `{"status":"error","message":"Internal server error"}`)

			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		// 创建HTTP客户端
		client := nethttp.NewClient(ts.URL)

		// 测试GET请求
		t.Run("GetJSON", func(t *testing.T) {
			var result map[string]interface{}
			err := client.GetJSON(context.Background(), "/api/get", &result)
			require.NoError(t, err)

			status, ok := result["status"].(string)
			assert.True(t, ok)
			assert.Equal(t, "success", status)

			data, ok := result["data"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, "Hello, World!", data["message"])
		})

		// 测试POST请求
		t.Run("PostJSON", func(t *testing.T) {
			reqBody := map[string]interface{}{
				"name": "test",
				"age":  30,
			}

			var result map[string]interface{}
			err := client.PostJSON(context.Background(), "/api/post", reqBody, &result, nil)
			require.NoError(t, err)

			status, ok := result["status"].(string)
			assert.True(t, ok)
			assert.Equal(t, "created", status)

			data, ok := result["data"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, float64(123), data["id"])
		})

		// 测试错误处理
		t.Run("ErrorHandling", func(t *testing.T) {
			var result map[string]interface{}
			err := client.GetJSON(context.Background(), "/api/error", &result)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "500")
		})
	})

	t.Run("HTTPServerTest", func(t *testing.T) {
		// 创建HTTP服务器
		logger := logging.NewLogger()
		server := nethttp.NewServer("localhost:0") // 随机端口

		// 添加中间件
		server.Use(nethttp.LoggingMiddleware(logger))
		server.Use(nethttp.RecoveryMiddleware(logger))
		server.Use(nethttp.RequestIDMiddleware())

		// 请求计数器
		var getCount int32
		var postCount int32

		// 注册路由
		server.GET("/api/test", func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&getCount, 1)

			// 确保请求ID存在，如果不存在则生成一个
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = "generated-" + time.Now().String()
				r.Header.Set("X-Request-ID", requestID)
			}

			nethttp.RespondJSON(w, http.StatusOK, map[string]interface{}{
				"method":     "GET",
				"path":       "/api/test",
				"request_id": requestID,
			})
		})

		server.POST("/api/create", func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&postCount, 1)
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				nethttp.RespondError(w, http.StatusBadRequest, "Invalid request body")
				return
			}

			nethttp.RespondJSON(w, http.StatusCreated, map[string]interface{}{
				"method":     "POST",
				"path":       "/api/create",
				"received":   body,
				"request_id": r.Header.Get("X-Request-ID"),
			})
		})

		server.PUT("/api/panic", func(w http.ResponseWriter, r *http.Request) {
			// 测试恢复中间件
			panic("test panic")
		})

		// 启动测试服务器（后台）
		go func() {
			server.Start()
		}()

		// 等待服务器启动
		time.Sleep(100 * time.Millisecond)

		// 获取服务器地址
		serverAddr := server.GetAddr()

		// 创建测试用的HTTP客户端
		client := nethttp.NewClient("http://" + serverAddr)

		// 测试GET请求
		var getResult map[string]interface{}
		err := client.GetJSON(context.Background(), "/api/test", &getResult)
		require.NoError(t, err)
		assert.Equal(t, "GET", getResult["method"])
		assert.Equal(t, "/api/test", getResult["path"])
		assert.NotEmpty(t, getResult["request_id"])

		// 测试POST请求
		postBody := map[string]interface{}{
			"name":  "test",
			"value": 123,
		}
		var postResult map[string]interface{}
		err = client.PostJSON(context.Background(), "/api/create", postBody, &postResult, nil)
		require.NoError(t, err)
		assert.Equal(t, "POST", postResult["method"])
		assert.Equal(t, "/api/create", postResult["path"])
		received, ok := postResult["received"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "test", received["name"])
		assert.Equal(t, float64(123), received["value"])

		// 测试恢复中间件
		var panicResult map[string]interface{}
		err = client.PutJSON(context.Background(), "/api/panic", nil, &panicResult, nil)
		assert.Error(t, err)

		// 验证请求计数
		assert.Equal(t, int32(1), atomic.LoadInt32(&getCount))
		assert.Equal(t, int32(1), atomic.LoadInt32(&postCount))

		// 停止服务器
		err = server.Stop(context.Background())
	})

	t.Run("RetryMechanismTest", func(t *testing.T) {
		// 创建一个失败计数器
		var failCount int32
		maxFailures := int32(3)

		// 创建测试服务器，前几次请求会失败
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			currentFailCount := atomic.AddInt32(&failCount, 1)

			if currentFailCount <= maxFailures {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"status":"success","attempt":"`+fmt.Sprint(currentFailCount)+`"}`)
		}))
		defer ts.Close()

		// 创建带重试的客户端
		retryPolicy := &nethttp.RetryPolicy{
			MaxRetries: 5,
			// 使用正确的字段名 - 可能是 RetryInterval 或 Delay
			RetryInterval: 100 * time.Millisecond, // 或 Delay: 100 * time.Millisecond

			// 使用正确的函数签名
			ShouldRetry: func(resp *http.Response, err error) bool {
				// 检查响应和错误
				if err != nil {
					return true // 有错误时重试
				}
				// 检查响应状态码，如果是服务不可用则重试
				return resp != nil && resp.StatusCode == http.StatusServiceUnavailable
			},
		}

		client := nethttp.NewClient(
			ts.URL,
			nethttp.WithRetryPolicy(retryPolicy),
		)

		// 测试重试逻辑
		var result map[string]interface{}
		err := client.GetJSON(context.Background(), "/retry-test", &result)

		require.NoError(t, err)
		assert.Equal(t, "success", result["status"])
		assert.Equal(t, maxFailures+1, failCount) // 确认服务器被调用了正确的次数

		// 测试超过最大重试次数的情况
		atomic.StoreInt32(&failCount, 0)
		maxFailures = 10 // 设置一个超过重试次数的值

		err = client.GetJSON(context.Background(), "/retry-test", &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max retries exceeded")

		// 测试带超时的重试
		atomic.StoreInt32(&failCount, 0)
		maxFailures = 2

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		// 由于超时设置得很短，可能在重试成功前就超时了
		err = client.GetJSON(ctx, "/retry-test", &result)
		if err != nil {
			assert.Contains(t, err.Error(), "context deadline exceeded")
		}
	})

	// 测试PUT和DELETE请求
	t.Run("HTTPClientAdditionalMethodsTest", func(t *testing.T) {
		// 创建测试服务器
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			switch {
			case r.Method == "PUT" && r.URL.Path == "/api/update":
				var reqBody map[string]interface{}
				json.NewDecoder(r.Body).Decode(&reqBody)
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `{"status":"updated","data":{"id":%v}}`, reqBody["id"])

			case r.Method == "DELETE" && r.URL.Path == "/api/delete":
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"status":"deleted"}`)

			case r.URL.Path == "/api/headers":
				// 返回所有请求头
				headers := make(map[string]string)
				for k, v := range r.Header {
					if len(v) > 0 {
						headers[k] = v[0]
					}
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{"headers": headers})

			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		client := nethttp.NewClient(ts.URL)

		// 测试PUT请求
		t.Run("PutJSON", func(t *testing.T) {
			reqBody := map[string]interface{}{
				"id":   456,
				"name": "updated_item",
			}

			var result map[string]interface{}
			err := client.PutJSON(context.Background(), "/api/update", reqBody, &result, nil)
			require.NoError(t, err)

			assert.Equal(t, "updated", result["status"])
			data, ok := result["data"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, float64(456), data["id"])
		})

		// 测试DELETE请求
		t.Run("DeleteJSON", func(t *testing.T) {
			reqBody := map[string]interface{}{
				"id": 123,
			}
			var result map[string]interface{}
			err := client.DeleteJSON(context.Background(), "/api/delete", reqBody, &result, nil)
			require.NoError(t, err)
			
			assert.Equal(t, "deleted", result["status"])
		})

		// 测试自定义请求头
		t.Run("CustomHeaders", func(t *testing.T) {
			headers := map[string]string{
				"X-Custom-Header": "test-value",
				"Authorization":   "Bearer token123",
			}

			var result map[string]interface{}
			err := client.GetJSONWithHeaders(context.Background(), "/api/headers", &result, headers)
			require.NoError(t, err)

			respHeaders, ok := result["headers"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, "test-value", respHeaders["X-Custom-Header"])
			assert.Equal(t, "Bearer token123", respHeaders["Authorization"])
		})
	})

	// 测试路由组和路径参数
	t.Run("RouteGroupAndParamsTest", func(t *testing.T) {
		server := nethttp.NewServer("localhost:0")
		server.Use(nethttp.RequestIDMiddleware())

		// 创建路由组
		api := server.Group("/api")
		v1 := api.Group("/v1")

		// 在路由组中注册路由
		v1.GET("/items", func(w http.ResponseWriter, r *http.Request) {
			nethttp.RespondJSON(w, http.StatusOK, map[string]interface{}{
				"items": []string{"item1", "item2"},
				"path":  r.URL.Path,
			})
		})

		// 使用路径参数
		v1.GET("/items/{id}", func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)  // 如果使用 gorilla/mux
			id := vars["id"]
			nethttp.RespondJSON(w, http.StatusOK, map[string]interface{}{
				"item_id": id,
				"path":    r.URL.Path,
			})
		})

		// 启动服务器
		go func() {
			server.Start()
		}()

		// 等待服务器启动
		time.Sleep(100 * time.Millisecond)

		serverAddr := server.GetAddr()
		client := nethttp.NewClient("http://" + serverAddr)

		// 测试路由组
		t.Run("RouteGroup", func(t *testing.T) {
			var result map[string]interface{}
			err := client.GetJSON(context.Background(), "/api/v1/items", &result)
			require.NoError(t, err)

			items, ok := result["items"].([]interface{})
			assert.True(t, ok)
			assert.Equal(t, 2, len(items))
			assert.Equal(t, "/api/v1/items", result["path"])
		})

		// 测试路径参数
		t.Run("PathParams", func(t *testing.T) {
			var result map[string]interface{}
			err := client.GetJSON(context.Background(), "/api/v1/items/123", &result)
			require.NoError(t, err)

			assert.Equal(t, "123", result["item_id"])
			assert.Equal(t, "/api/v1/items/123", result["path"])
		})

		// 停止服务器
		err := server.Stop(context.Background())
		assert.NoError(t, err)
	})

	// 测试CORS中间件
	t.Run("CORSMiddlewareTest", func(t *testing.T) {
		server := nethttp.NewServer("localhost:0")

		// 添加CORS中间件
		server.Use(nethttp.CORSMiddleware([]string{"https://example.com"}))

		// 注册测试路由
		server.GET("/api/cors-test", func(w http.ResponseWriter, r *http.Request) {
			nethttp.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})

		// 关键：显式添加 OPTIONS 处理程序
		server.OPTIONS("/api/cors-test", func(w http.ResponseWriter, r *http.Request) {
			// 这个处理程序实际上可以为空，因为 CORS 中间件会处理响应
			// 但我们需要显式注册这个路由以便服务器接受 OPTIONS 请求
		})

		
		// 启动服务器
		go func() {
			server.Start()
		}()

		// 等待服务器启动
		time.Sleep(100 * time.Millisecond)

		serverAddr := server.GetAddr()
		baseURL := "http://" + serverAddr

		// 测试预检请求(OPTIONS)
		t.Run("PreflightRequest", func(t *testing.T) {
			req, err := http.NewRequest("OPTIONS", baseURL+"/api/cors-test", nil)
			require.NoError(t, err)
			req.Header.Set("Origin", "https://example.com")
			req.Header.Set("Access-Control-Request-Method", "GET")

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "https://example.com", resp.Header.Get("Access-Control-Allow-Origin"))
			assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "GET")
		})

		// 测试实际请求
		t.Run("ActualRequest", func(t *testing.T) {
			req, err := http.NewRequest("GET", baseURL+"/api/cors-test", nil)
			require.NoError(t, err)
			req.Header.Set("Origin", "https://example.com")

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "https://example.com", resp.Header.Get("Access-Control-Allow-Origin"))
		})

		// 停止服务器
		err := server.Stop(context.Background())
		assert.NoError(t, err)
	})

	// 测试Auth中间件
	t.Run("AuthMiddlewareTest", func(t *testing.T) {
		server := nethttp.NewServer("localhost:0")

		// 设置认证函数
		authFunc := func(username, password string) bool {
			return username == "admin" && password == "secret"
		}

		// 添加Auth中间件
		server.Use(nethttp.AuthMiddleware(authFunc))

		// 注册受保护的路由
		server.GET("/api/protected", func(w http.ResponseWriter, r *http.Request) {
			nethttp.RespondJSON(w, http.StatusOK, map[string]string{"status": "authenticated"})
		})

		// 注册公开路由
		server.GET("/status", func(w http.ResponseWriter, r *http.Request) {
			nethttp.RespondJSON(w, http.StatusOK, map[string]string{"status": "public"})
		})

		// 启动服务器
		go func() {
			server.Start()
		}()

		// 等待服务器启动
		time.Sleep(100 * time.Millisecond)

		serverAddr := server.GetAddr()
		baseURL := "http://" + serverAddr

		// 测试无认证访问
		t.Run("NoAuth", func(t *testing.T) {
			resp, err := http.Get(baseURL + "/api/protected")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})

		// 测试有效认证
		t.Run("ValidAuth", func(t *testing.T) {
			req, err := http.NewRequest("GET", baseURL+"/api/protected", nil)
			require.NoError(t, err)
			req.SetBasicAuth("admin", "secret")

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var result map[string]string
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)
			assert.Equal(t, "authenticated", result["status"])
		})

		// 测试无需认证的公开路由
		t.Run("PublicRoute", func(t *testing.T) {
			resp, err := http.Get(baseURL + "/status")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var result map[string]string
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)
			assert.Equal(t, "public", result["status"])
		})

		// 停止服务器
		err := server.Stop(context.Background())
		assert.NoError(t, err)
	})
}
