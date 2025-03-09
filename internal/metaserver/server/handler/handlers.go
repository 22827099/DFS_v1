package handler

import (
	"net/http"
	"path"
	"strings"

	"github.com/22827099/DFS_v1/common/errors"
	"github.com/22827099/DFS_v1/common/logging"
	httplib "github.com/22827099/DFS_v1/common/network/http"
	"github.com/22827099/DFS_v1/internal/metaserver/core"
	"github.com/22827099/DFS_v1/internal/metaserver/core/models_share"
)

// Handlers 包含所有API处理器
type Handlers struct {
	core   *core.MetaCore
	logger logging.Logger
}

// NewHandlers 创建新的处理器集合
func NewHandlers(core *core.MetaCore, logger logging.Logger) *Handlers {
	return &Handlers{
		core:   core,
		logger: logger,
	}
}

// GetMetadata 获取路径的元数据
func (h *Handlers) GetMetadata(c *httplib.Context) {
	pathParam := c.Param("path")
	normalized := normalizePath(pathParam)

	ctx := c.Request.Context()
	meta, err := h.core.Meta.GetMetadata(ctx, normalized)
	if err != nil {
		h.logger.Error("获取元数据失败: %v, 路径: %s", err, normalized)
		httplib.WriteError(c.Response, http.StatusInternalServerError, errors.Message(err), errors.Code(err))
		return
	}

	if meta == nil || !meta.Exists {
		httplib.WriteError(c.Response, http.StatusNotFound, "路径不存在", 404)
		return
	}

	httplib.WriteSuccess(c.Response, meta)
}

// CreateOrUpdateMetadata 创建或更新元数据
func (h *Handlers) CreateOrUpdateMetadata(c *httplib.Context) {
	pathParam := c.Param("path")
	normalized := normalizePath(pathParam)

	var metaReq models_share.MetadataRequest
	if err := c.BindJSON(&metaReq); err != nil {
		httplib.WriteError(c.Response, http.StatusBadRequest, "无效的请求数据", 400)
		return
	}

	ctx := c.Request.Context()
	result, err := h.core.Meta.CreateOrUpdate(ctx, normalized, &metaReq)
	if err != nil {
		h.logger.Error("创建/更新元数据失败: %v, 路径: %s", err, normalized)
		httplib.WriteError(c.Response, http.StatusInternalServerError, errors.Message(err), errors.Code(err))
		return
	}

	httplib.WriteSuccess(c.Response, result)
}

// DeleteMetadata 删除元数据
func (h *Handlers) DeleteMetadata(c *httplib.Context) {
	pathParam := c.Param("path")
	normalized := normalizePath(pathParam)

	recursive := c.QueryParam("recursive") == "true"

	ctx := c.Request.Context()
	err := h.core.Meta.Delete(ctx, normalized, recursive)
	if err != nil {
		h.logger.Error("删除元数据失败: %v, 路径: %s", err, normalized)
		httplib.WriteError(c.Response, http.StatusInternalServerError, errors.Message(err), errors.Code(err))
		return
	}

	httplib.WriteSuccess(c.Response, map[string]interface{}{
		"success": true,
		"path":    normalized,
	})
}

// ListDirectory 列出目录内容
func (h *Handlers) ListDirectory(c *httplib.Context) {
	pathParam := c.Param("path")
	normalized := normalizePath(pathParam)

	sortBy := c.QueryParam("sort")
	sortOrder := c.QueryParam("order")

	ctx := c.Request.Context()
	entries, err := h.core.Meta.ListDirectory(ctx, normalized, WithSort(sortBy, sortOrder))
	if err != nil {
		h.logger.Error("列出目录内容失败: %v, 路径: %s", err, normalized)
		httplib.WriteError(c.Response, http.StatusInternalServerError, errors.Message(err), errors.Code(err))
		return
	}

	httplib.WriteSuccess(c.Response, entries)
}

// MoveMetadata 移动文件或目录
func (h *Handlers) MoveMetadata(c *httplib.Context) {
	// 实现移动功能
}

// CopyMetadata 复制文件或目录
func (h *Handlers) CopyMetadata(c *httplib.Context) {
	// 实现复制功能
}

// BatchOperation 批量操作
func (h *Handlers) BatchOperation(c *httplib.Context) {
	// 实现批量操作功能
}

// GetClusterStatus 获取集群状态
func (h *Handlers) GetClusterStatus(c *httplib.Context) {
	ctx := c.Request.Context()
	status := h.core.Cluster.GetStatus(ctx)
	httplib.WriteSuccess(c.Response, status)
}

// ListClusterNodes 列出集群节点
func (h *Handlers) ListClusterNodes(c *httplib.Context) {
	nodes := h.core.Cluster.GetNodes()
	httplib.WriteSuccess(c.Response, nodes)
}

// StartRebalance 开始负载均衡
func (h *Handlers) StartRebalance(c *httplib.Context) {
	// 实现负载均衡触发功能
}

// GetSystemStats 获取系统统计信息
func (h *Handlers) GetSystemStats(c *httplib.Context) {
	// 实现系统统计功能
}

// HealthCheck 健康检查
func (h *Handlers) HealthCheck(c *httplib.Context) {
	ctx := c.Request.Context()
	health := h.core.HealthCheck(ctx)

	status := http.StatusOK
	if !health.Healthy {
		status = http.StatusServiceUnavailable
	}

	c.Response.WriteHeader(status)
	httplib.WriteJSON(c.Response, health)
}

// SystemStatus 系统状态
func (h *Handlers) SystemStatus(c *httplib.Context) {
	httplib.WriteSuccess(c.Response, map[string]interface{}{
		"status":  "running",
		"version": "1.0.0",
		"uptime":  h.core.Uptime().String(),
	})
}

// AuthHandler 身份验证处理
func (h *Handlers) AuthHandler(username, password string) bool {
	// 实现身份验证逻辑
	return true // 简化示例
}

// CreateUser 创建用户
func (h *Handlers) CreateUser(c *httplib.Context) {
	// 实现用户创建功能
}

// GetUser 获取用户信息
func (h *Handlers) GetUser(c *httplib.Context) {
	// 实现获取用户信息功能
}

// UpdateUser 更新用户信息
func (h *Handlers) UpdateUser(c *httplib.Context) {
	// 实现更新用户信息功能
}

// 辅助函数

// normalizePath 规范化路径
func normalizePath(p string) string {
	if p == "" {
		return "/"
	}

	// 确保路径以/开头
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	// 清理路径
	return path.Clean(p)
}

// WithSort 创建排序选项
func WithSort(field string, order string) core.ListOption {
	return func(opts *core.ListOptions) {
		if field != "" {
			opts.SortBy = field
			if order == "desc" {
				opts.SortOrder = "desc"
			} else {
				opts.SortOrder = "asc"
			}
		}
	}
}
