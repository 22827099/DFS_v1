package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	networkHttp "github.com/22827099/DFS_v1/common/network/http"
)

func setupTestServer() (*httptest.Server, *http.ServeMux) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	return server, mux
}

func TestNewClient(t *testing.T) {
	client := networkHttp.NewClient("http://example.com")
	if client == nil {
		t.Fatalf("NewClient: 返回nil")
	}

	client = networkHttp.NewClient(
		"http://example.com",
		networkHttp.WithClientTimeout(10*time.Second),
		networkHttp.WithRetryPolicy(2, 200*time.Millisecond),
	)
	if client == nil {
		t.Fatalf("NewClient: 使用选项时返回nil")
	}
}

func TestClient_GetJSON(t *testing.T) {
	server, mux := setupTestServer()
	defer server.Close()

	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Client.GetJSON: 期望GET请求，得到%s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "成功"})
	})

	client := networkHttp.NewClient(server.URL)

	var result map[string]string
	err := client.GetJSON(context.Background(), "/api/test", &result)
	if err != nil {
		t.Fatalf("Client.GetJSON: 返回错误: %v", err)
	}

	if result["message"] != "成功" {
		t.Errorf("Client.GetJSON: 期望message为'成功'，得到'%s'", result["message"])
	}
}

func TestClient_PostJSON(t *testing.T) {
	server, mux := setupTestServer()
	defer server.Close()

	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Client.PostJSON: 期望POST请求，得到%s", r.Method)
		}

		var requestBody map[string]string
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Client.PostJSON: 无法解析请求体: %v", err)
		}
		defer r.Body.Close()

		if requestBody["key"] != "value" {
			t.Errorf("Client.PostJSON: 期望key为'value'，得到'%s'", requestBody["key"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "已创建"})
	})

	client := networkHttp.NewClient(server.URL)

	requestBody := map[string]string{"key": "value"}
	var result map[string]string
	err := client.PostJSON(context.Background(), "/api/test", requestBody, &result)
	if err != nil {
		t.Fatalf("Client.PostJSON: 返回错误: %v", err)
	}

	if result["status"] != "已创建" {
		t.Errorf("Client.PostJSON: 期望status为'已创建'，得到'%s'", result["status"])
	}
}

func TestClient_PutJSON(t *testing.T) {
	server, mux := setupTestServer()
	defer server.Close()

	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Client.PutJSON: 期望PUT请求，得到%s", r.Method)
		}

		var requestBody map[string]string
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Client.PutJSON: 无法解析请求体: %v", err)
		}
		defer r.Body.Close()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "已更新"})
	})

	client := networkHttp.NewClient(server.URL)

	requestBody := map[string]string{"key": "updated"}
	var result map[string]string
	err := client.PutJSON(context.Background(), "/api/test", requestBody, &result)
	if err != nil {
		t.Fatalf("Client.PutJSON: 返回错误: %v", err)
	}

	if result["status"] != "已更新" {
		t.Errorf("Client.PutJSON: 期望status为'已更新'，得到'%s'", result["status"])
	}
}

func TestClient_DeleteJSON(t *testing.T) {
	server, mux := setupTestServer()
	defer server.Close()

	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Client.DeleteJSON: 期望DELETE请求，得到%s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "已删除"})
	})

	client := networkHttp.NewClient(server.URL)

	var result map[string]string
	err := client.DeleteJSON(context.Background(), "/api/test", &result)
	if err != nil {
		t.Fatalf("Client.DeleteJSON: 返回错误: %v", err)
	}

	if result["status"] != "已删除" {
		t.Errorf("Client.DeleteJSON: 期望status为'已删除'，得到'%s'", result["status"])
	}
}

func TestClient_Retry(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// 前两次请求失败，第三次成功
		if requestCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "成功"})
	}))
	defer server.Close()

	client := networkHttp.NewClient(
		server.URL,
		networkHttp.WithRetryPolicy(3, 10*time.Millisecond),
	)

	var result map[string]string
	err := client.GetJSON(context.Background(), "/", &result)
	if err != nil {
		t.Fatalf("Client.Retry: 带重试的GetJSON返回错误: %v", err)
	}

	if result["status"] != "成功" {
		t.Errorf("Client.Retry: 期望status为'成功'，得到'%s'", result["status"])
	}

	if requestCount != 3 {
		t.Errorf("Client.Retry: 由于重试策略，期望3次请求，得到%d次", requestCount)
	}
}
