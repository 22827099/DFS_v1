package server

import (
	"encoding/json"
	"io"
	"net/http"
	"path"
	"strings"
	"time"
	"fmt"

	"github.com/22827099/DFS_v1/common/errors"
	nethttp "github.com/22827099/DFS_v1/common/network/http"
	"github.com/22827099/DFS_v1/internal/metaserver/core/metadata"
	"github.com/gorilla/mux"
)

// handleHealthCheck 处理健康检查请求
func (s *MetadataServer) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "running",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
	}

	nethttp.RespondJSON(w, http.StatusOK, status)
}

// handleGetFileInfo 获取文件信息
func (s *MetadataServer) handleGetFileInfo(w http.ResponseWriter, r *http.Request) {
	filePath := extractPath(r)
	if filePath == "" {
		nethttp.RespondError(w, http.StatusBadRequest, "无效的文件路径")
		return
	}

	fileInfo, err := s.metaStore.GetFileInfo(r.Context(), filePath)
	if err != nil {
		handleError(w, err)
		return
	}

	nethttp.RespondJSON(w, http.StatusOK, fileInfo)
}

// handleCreateFile 创建文件
func (s *MetadataServer) handleCreateFile(w http.ResponseWriter, r *http.Request) {
	filePath := extractPath(r)
	if filePath == "" {
		nethttp.RespondError(w, http.StatusBadRequest, "无效的文件路径")
		return
	}

	var fileInfo metadata.FileInfo
	if err := json.NewDecoder(r.Body).Decode(&fileInfo); err != nil {
		nethttp.RespondError(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	defer r.Body.Close()

	// 设置文件路径
	fileInfo.Path = filePath

	// 创建文件元数据
	result, err := s.metaStore.CreateFile(r.Context(), fileInfo)
	if err != nil {
		handleError(w, err)
		return
	}

	nethttp.RespondJSON(w, http.StatusCreated, result)
}

// handleUpdateFile 更新文件信息
func (s *MetadataServer) handleUpdateFile(w http.ResponseWriter, r *http.Request) {
	filePath := extractPath(r)
	if filePath == "" {
		nethttp.RespondError(w, http.StatusBadRequest, "无效的文件路径")
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		nethttp.RespondError(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	defer r.Body.Close()

	// 更新文件元数据
	result, err := s.metaStore.UpdateFile(r.Context(), filePath, updates)
	if err != nil {
		handleError(w, err)
		return
	}

	nethttp.RespondJSON(w, http.StatusOK, result)
}

// handleDeleteFile 删除文件
func (s *MetadataServer) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	filePath := extractPath(r)
	if filePath == "" {
		nethttp.RespondError(w, http.StatusBadRequest, "无效的文件路径")
		return
	}

	err := s.metaStore.DeleteFile(r.Context(), filePath)
	if err != nil {
		handleError(w, err)
		return
	}

	nethttp.RespondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// handleListDirectory 列出目录内容
func (s *MetadataServer) handleListDirectory(w http.ResponseWriter, r *http.Request) {
	dirPath := extractPath(r)
	if dirPath == "" {
		nethttp.RespondError(w, http.StatusBadRequest, "无效的目录路径")
		return
	}

	// 获取可选参数
	recursive := r.URL.Query().Get("recursive") == "true"
	limit := 100 // 默认限制，实际应该从查询参数获取

	entries, err := s.metaStore.ListDirectory(r.Context(), dirPath, recursive, limit)
	if err != nil {
		handleError(w, err)
		return
	}

	nethttp.RespondJSON(w, http.StatusOK, entries)
}

// handleCreateDirectory 创建目录
func (s *MetadataServer) handleCreateDirectory(w http.ResponseWriter, r *http.Request) {
	dirPath := extractPath(r)
	if dirPath == "" {
		nethttp.RespondError(w, http.StatusBadRequest, "无效的目录路径")
		return
	}

	var dirInfo metadata.DirectoryInfo
	body, err := io.ReadAll(r.Body)
	if err != nil {
		// 处理错误
		nethttp.RespondError(w, http.StatusBadRequest, fmt.Sprintf("读取请求体失败: %v", err))
		return
	}
	defer r.Body.Close()
	
	// 尝试解析请求体，但允许为空
	if len(body) > 0 {
		if err := json.Unmarshal(body, &dirInfo); err != nil {
			nethttp.RespondError(w, http.StatusBadRequest, "无效的请求体")
			return
		}
	}

	// 设置目录路径
	dirInfo.Path = dirPath

	// 创建目录
	result, err := s.metaStore.CreateDirectory(r.Context(), dirInfo)
	if err != nil {
		handleError(w, err)
		return
	}

	nethttp.RespondJSON(w, http.StatusCreated, result)
}

// handleDeleteDirectory 删除目录
func (s *MetadataServer) handleDeleteDirectory(w http.ResponseWriter, r *http.Request) {
	dirPath := extractPath(r)
	if dirPath == "" {
		nethttp.RespondError(w, http.StatusBadRequest, "无效的目录路径")
		return
	}

	// 获取可选参数
	recursive := r.URL.Query().Get("recursive") == "true"

	err := s.metaStore.DeleteDirectory(r.Context(), dirPath, recursive)
	if err != nil {
		handleError(w, err)
		return
	}

	nethttp.RespondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// handleListNodes 列出集群节点
func (s *MetadataServer) handleListNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.cluster.ListNodes(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}

	nethttp.RespondJSON(w, http.StatusOK, nodes)
}

// handleGetNodeInfo 获取节点信息
func (s *MetadataServer) handleGetNodeInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]
	if nodeID == "" {
		nethttp.RespondError(w, http.StatusBadRequest, "节点ID不能为空")
		return
	}

	node, err := s.cluster.GetNodeInfo(r.Context(), nodeID)
	if err != nil {
		handleError(w, err)
		return
	}

	nethttp.RespondJSON(w, http.StatusOK, node)
}

// handleGetLeader 获取当前集群领导者信息
func (s *MetadataServer) handleGetLeader(w http.ResponseWriter, r *http.Request) {
	leader, err := s.cluster.GetLeader(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}

	nethttp.RespondJSON(w, http.StatusOK, leader)
}

// handleServerStatus 获取服务器状态
func (s *MetadataServer) handleServerStatus(w http.ResponseWriter, r *http.Request) {
	isLeader := s.cluster.IsLeader()
	
	status := map[string]interface{}{
		"id":          s.config.NodeID,
		"uptime":      "1h30m", // 应该计算实际运行时间
		"is_leader":   isLeader,
		"connections": 42,      // 示例值，实际应该从连接池获取
		"version":     "1.0.0",
	}

	nethttp.RespondJSON(w, http.StatusOK, status)
}

// 辅助函数

// extractPath 从请求中提取文件或目录的路径
func extractPath(r *http.Request) string {
	pathParam := mux.Vars(r)["path"]
	if pathParam == "" {
		return ""
	}
	
	// 确保路径以/开头
	if !strings.HasPrefix(pathParam, "/") {
		pathParam = "/" + pathParam
	}
	
	// 规范化路径
	return path.Clean(pathParam)
}

// handleError 处理错误并返回适当的HTTP响应
func handleError(w http.ResponseWriter, err error) {
	// 首先尝试将err转换为我们的自定义错误
	statusCode := http.StatusInternalServerError
	errorMsg := "内部服务器错误"
	
	if e, ok := err.(*errors.Error); ok {
		// 根据错误代码设置HTTP状态码
		switch e.Code {
		case errors.NotFound:
			statusCode = http.StatusNotFound
			errorMsg = "资源未找到"
		case errors.InvalidArgument:
			statusCode = http.StatusBadRequest
			errorMsg = "参数无效"
		case errors.PermissionDenied:
			statusCode = http.StatusForbidden
			errorMsg = "权限不足"
		case errors.AlreadyExists:
			statusCode = http.StatusConflict
			errorMsg = "资源已存在"
		case errors.Unauthenticated:
			statusCode = http.StatusUnauthorized
			errorMsg = "认证失败"
		}
		
		// 使用错误消息（如果有）
		if e.Message != "" {
			errorMsg = e.Message
		}
	}
	
	// 返回错误响应
	nethttp.RespondError(w, statusCode, errorMsg)
}
