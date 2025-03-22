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

func TestE2E(t *testing.T) {
	// 如果测试时间太长，可以跳过
	if testing.Short() {
		t.Skip("跳过耗时的端到端测试")
	}

	// 创建配置
	cfg := &config.SystemConfig{
		NodeID: "test-node",
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 18080, // 使用指定端口
		},
	}

	// 创建服务器
	metaServer, err := server.NewServer(cfg)
	require.NoError(t, err)

	// 启动服务器
	err = metaServer.Start()
	require.NoError(t, err)

	// 确保测试结束后停止服务器
	defer func() {
		err := metaServer.Stop()
		assert.NoError(t, err)
	}()

	// 给服务器一些启动时间
	time.Sleep(500 * time.Millisecond)

	baseURL := fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port)

	t.Run("CreateDirectoryFlow", func(t *testing.T) {
		// 创建目录
		dirReq := map[string]interface{}{
			"name": "test_dir",
		}
		reqBody, err := json.Marshal(dirReq)
		require.NoError(t, err)

		// 发送HTTP请求创建目录
		resp, err := http.Post(
			baseURL+"/api/v1/directories/test_dir",
			"application/json",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		// 验证目录创建成功
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "test_dir", result["name"])
		assert.Equal(t, "/test_dir/", result["path"])
	})

	t.Run("CreateFileFlow", func(t *testing.T) {
		// 创建文件
		fileReq := map[string]interface{}{
			"name":      "test.txt",
			"size":      1024,
			"mime_type": "text/plain",
		}
		reqBody, err := json.Marshal(fileReq)
		require.NoError(t, err)

		// 发送HTTP请求创建文件
		resp, err := http.Post(
			baseURL+"/api/v1/files/test.txt",
			"application/json",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		// 验证文件创建成功
		var fileResult map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&fileResult)
		require.NoError(t, err)
		assert.Equal(t, "test.txt", fileResult["name"])
		assert.Equal(t, float64(1024), fileResult["size"])
		assert.Equal(t, "text/plain", fileResult["mime_type"])
	})

	t.Run("UpdateFileFlow", func(t *testing.T) {
		// 更新文件信息
		updateReq := map[string]interface{}{
			"size":      2048,
			"mime_type": "application/octet-stream",
		}
		reqBody, err := json.Marshal(updateReq)
		require.NoError(t, err)

		// 创建PUT请求
		req, err := http.NewRequest(
			http.MethodPut,
			baseURL+"/api/v1/files/test.txt",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// 发送请求
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 验证文件更新成功
		var updateResult map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&updateResult)
		require.NoError(t, err)
		assert.Equal(t, "test.txt", updateResult["name"])
		assert.Equal(t, float64(2048), updateResult["size"])
		assert.Equal(t, "application/octet-stream", updateResult["mime_type"])
	})

	t.Run("ListDirectoryFlow", func(t *testing.T) {
		// 在测试目录下创建文件
		fileReq := map[string]interface{}{
			"name":      "dir_file.txt",
			"size":      512,
			"mime_type": "text/plain",
		}
		reqBody, err := json.Marshal(fileReq)
		require.NoError(t, err)

		// 发送HTTP请求在目录中创建文件
		resp, err := http.Post(
			baseURL+"/api/v1/files/test_dir/dir_file.txt",
			"application/json",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err)
		resp.Body.Close()

		// 发送GET请求获取目录列表
		resp, err = http.Get(baseURL + "/api/v1/directories/test_dir")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 解析目录列表响应
		var listResult map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&listResult)
		require.NoError(t, err)

		// 验证目录内容
		entries, ok := listResult["entries"].([]interface{})
		require.True(t, ok)
		assert.True(t, len(entries) > 0, "目录应该包含至少一个文件")

		// 检查是否包含我们创建的文件
		foundFile := false
		for _, entry := range entries {
			item, ok := entry.(map[string]interface{})
			if ok && item["name"] == "dir_file.txt" {
				foundFile = true
				break
			}
		}
		assert.True(t, foundFile, "目录列表应包含创建的文件")
	})

	t.Run("DeleteFileFlow", func(t *testing.T) {
		// 创建DELETE请求
		req, err := http.NewRequest(
			http.MethodDelete,
			baseURL+"/api/v1/files/test.txt",
			nil,
		)
		require.NoError(t, err)

		// 发送请求
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 尝试再次获取该文件，应该返回404
		resp, err = http.Get(baseURL + "/api/v1/files/test.txt")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("DeleteDirectoryFlow", func(t *testing.T) {
		// 创建递归删除的DELETE请求
		req, err := http.NewRequest(
			http.MethodDelete,
			baseURL+"/api/v1/directories/test_dir?recursive=true",
			nil,
		)
		require.NoError(t, err)

		// 发送请求
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 尝试再次获取该目录，应该返回404
		resp, err = http.Get(baseURL + "/api/v1/directories/test_dir")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("ErrorHandlingFlow", func(t *testing.T) {
		// 测试无效请求体
		resp, err := http.Post(
			baseURL+"/api/v1/files/invalid.txt",
			"application/json",
			bytes.NewReader([]byte("这不是有效的JSON")),
		)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// 测试创建已存在的资源
		dirReq := map[string]interface{}{
			"name": "duplicate_dir",
		}
		reqBody, _ := json.Marshal(dirReq)

		// 第一次创建
		resp, _ = http.Post(
			baseURL+"/api/v1/directories/duplicate_dir",
			"application/json",
			bytes.NewReader(reqBody),
		)
		resp.Body.Close()

		// 第二次创建应失败
		resp, err = http.Post(
			baseURL+"/api/v1/directories/duplicate_dir",
			"application/json",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		// 清理创建的目录
		req, _ := http.NewRequest(
			http.MethodDelete,
			baseURL+"/api/v1/directories/duplicate_dir",
			nil,
		)
		client := &http.Client{}
		client.Do(req)
	})

	t.Run("HealthCheckFlow", func(t *testing.T) {
		// 发送健康检查请求
		resp, err := http.Get(baseURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 解析响应
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		// 验证健康检查结果
		assert.Equal(t, "running", result["status"])
		assert.NotEmpty(t, result["timestamp"])
		assert.NotEmpty(t, result["version"])
	})

	t.Run("SearchAndFilterFlow", func(t *testing.T) {
	// 先创建测试数据
	createTestFiles := func() {
		// 创建多个测试文件
		fileTypes := []string{"text/plain", "application/pdf", "image/jpeg"}
		fileSizes := []int{1024, 2048, 4096}
		
		for i, fileType := range fileTypes {
			fileReq := map[string]interface{}{
				"name":      fmt.Sprintf("search_file%d.txt", i),
				"size":      fileSizes[i],
				"mime_type": fileType,
			}
			reqBody, _ := json.Marshal(fileReq)
			resp, _ := http.Post(
				baseURL+fmt.Sprintf("/api/v1/files/search_file%d.txt", i),
				"application/json",
				bytes.NewReader(reqBody),
			)
			resp.Body.Close()
		}
	}
	createTestFiles()
	
	// 按类型搜索/过滤
	resp, err := http.Get(baseURL + "/api/v1/files?mime_type=text/plain")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var searchResult map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&searchResult)
	require.NoError(t, err)
	
	// 验证搜索结果
	files, ok := searchResult["files"].([]interface{})
	require.True(t, ok)
	
	textFiles := 0
	for _, file := range files {
		fileMap := file.(map[string]interface{})
		if fileMap["mime_type"] == "text/plain" {
			textFiles++
		}
	}
	assert.True(t, textFiles > 0, "应该至少找到一个text/plain类型的文件")
	
	// 清理测试文件
	cleanupTestFiles := func() {
		client := &http.Client{}
		for i := 0; i < 3; i++ {
			req, _ := http.NewRequest(
				http.MethodDelete,
				baseURL+fmt.Sprintf("/api/v1/files/search_file%d.txt", i),
				nil,
			)
			client.Do(req)
		}
	}
	cleanupTestFiles()
})

	t.Run("BatchOperationsFlow", func(t *testing.T) {
		// 创建测试目录
		batchDir := map[string]interface{}{
			"name": "batch_dir",
		}
		reqBody, _ := json.Marshal(batchDir)
		resp, _ := http.Post(
			baseURL+"/api/v1/directories/batch_dir",
			"application/json",
			bytes.NewReader(reqBody),
		)
		resp.Body.Close()
		
		// 批量创建测试文件
		batchFilesReq := map[string]interface{}{
			"files": []map[string]interface{}{
				{
					"name": "batch1.txt",
					"size": 1024,
					"mime_type": "text/plain",
				},
				{
					"name": "batch2.txt",
					"size": 2048,
					"mime_type": "text/plain",
				},
			},
		}
		reqBody, err := json.Marshal(batchFilesReq)
		require.NoError(t, err)
		
		// 发送批量创建请求
		resp, err = http.Post(
			baseURL+"/api/v1/batch/files/batch_dir",
			"application/json",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		// 验证批量创建结果
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		var batchResult map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&batchResult)
		require.NoError(t, err)
		
		results, ok := batchResult["results"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 2, len(results), "应该创建2个文件")
		
		// 检查目录内容
		resp, err = http.Get(baseURL + "/api/v1/directories/batch_dir")
		require.NoError(t, err)
		defer resp.Body.Close()
		
		var dirContents map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&dirContents)
		require.NoError(t, err)
		
		entries, _ := dirContents["entries"].([]interface{})
		assert.Equal(t, 2, len(entries), "目录应包含2个文件")
		
		// 清理测试数据
		req, _ := http.NewRequest(
			http.MethodDelete,
			baseURL+"/api/v1/directories/batch_dir?recursive=true",
			nil,
		)
		client := &http.Client{}
		client.Do(req)
	})

	t.Run("PermissionsFlow", func(t *testing.T) {
        // 创建测试文件
        fileReq := map[string]interface{}{
            "name":      "permission_test.txt",
            "size":      1024,
            "mime_type": "text/plain",
        }
        reqBody, _ := json.Marshal(fileReq)
        resp, _ := http.Post(
            baseURL+"/api/v1/files/permission_test.txt",
            "application/json",
            bytes.NewReader(reqBody),
        )
        resp.Body.Close()
        
	// 更新文件权限
	permReq := map[string]interface{}{
		"owner": "user1",
		"group": "group1",
		"mode":  0644,
	}
	reqBody, err := json.Marshal(permReq)
	require.NoError(t, err)
	
	// 发送权限更新请求
	req, err := http.NewRequest(
		http.MethodPatch,
		baseURL+"/api/v1/files/permission_test.txt/permissions",
		bytes.NewReader(reqBody),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	// 验证权限更新结果
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// 获取并验证更新后的权限
	resp, err = http.Get(baseURL + "/api/v1/files/permission_test.txt")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	var fileInfo map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&fileInfo)
	require.NoError(t, err)
	
	assert.Equal(t, "user1", fileInfo["owner"])
	assert.Equal(t, "group1", fileInfo["group"])
	assert.Equal(t, float64(0644), fileInfo["mode"])
	
	// 清理测试文件
	req, _ = http.NewRequest(
		http.MethodDelete,
		baseURL+"/api/v1/files/permission_test.txt",
		nil,
	)
	client.Do(req)
	})

	t.Run("ServerStatusFlow", func(t *testing.T) {
        // 获取服务器状态信息
        resp, err := http.Get(baseURL + "/api/v1/admin/status")
        require.NoError(t, err)
        defer resp.Body.Close()
        
        assert.Equal(t, http.StatusOK, resp.StatusCode)
        
        var statusInfo map[string]interface{}
        err = json.NewDecoder(resp.Body).Decode(&statusInfo)
        require.NoError(t, err)
        
        // 验证状态信息包含关键指标
        assert.Contains(t, statusInfo, "uptime")
        assert.Contains(t, statusInfo, "version")
        assert.Contains(t, statusInfo, "metrics")
        
        metrics, ok := statusInfo["metrics"].(map[string]interface{})
        require.True(t, ok)
        assert.Contains(t, metrics, "memory_usage")
        assert.Contains(t, metrics, "cpu_usage")
        assert.Contains(t, metrics, "open_connections")
        assert.Contains(t, metrics, "request_count")
    })
}

