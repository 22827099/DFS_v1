package metaserver_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/22827099/DFS_v1/common/config"
    "github.com/22827099/DFS_v1/internal/metaserver/server"
    "github.com/22827099/DFS_v1/internal/metaserver/server/api/v1"
)

func TestAPI(t *testing.T) {
    t.Run("FilesAPITest", func(t *testing.T) {
        // 创建内存存储
        store, err := server.NewMemoryStore()
        require.NoError(t, err)
        require.NoError(t, store.Initialize())

        // 创建文件API处理器
        filesAPI := v1.NewFilesAPI(store)
        
        // 测试创建文件
        fileReq := v1.FileRequest{
            Name:     "test.txt",
            Size:     1024,
            MimeType: "text/plain",
        }
        body, err := json.Marshal(fileReq)
        require.NoError(t, err)

        req := httptest.NewRequest(http.MethodPost, "/api/v1/files/test.txt", bytes.NewReader(body))
        w := httptest.NewRecorder()
        
        // 手动设置路径参数，正常情况下由路由器设置
        // 在实际集成测试中，可以使用httprouter或gorilla/mux创建完整路由
        ctx := req.Context()
        req = req.WithContext(ctx)
        
        filesAPI.CreateFile(w, req)
        resp := w.Result()
        
        assert.Equal(t, http.StatusCreated, resp.StatusCode)
        
        // 测试获取文件信息
        req = httptest.NewRequest(http.MethodGet, "/api/v1/files/test.txt", nil)
        w = httptest.NewRecorder()
        
        filesAPI.GetFileInfo(w, req)
        resp = w.Result()
        
        assert.Equal(t, http.StatusOK, resp.StatusCode)
        
        var result map[string]interface{}
        err = json.NewDecoder(resp.Body).Decode(&result)
        require.NoError(t, err)
        assert.Equal(t, "test.txt", result["name"])
    })
}