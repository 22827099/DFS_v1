package v1

import (
    "net/http"
    "encoding/json"
    "io"
    
    "github.com/22827099/DFS_v1/common/errors"
    "github.com/22827099/DFS_v1/internal/metaserver/core/metadata"
    "github.com/22827099/DFS_v1/internal/metaserver/server/api"
    "github.com/gorilla/mux"
    nethttp "github.com/22827099/DFS_v1/common/network/http"
    "github.com/22827099/DFS_v1/common/utils"
)

// DirectoriesAPI 处理目录相关的API请求
type DirectoriesAPI struct {
    store metadata.Store
}

// NewDirectoriesAPI 创建目录API处理器
func NewDirectoriesAPI(store metadata.Store) *DirectoriesAPI {
    return &DirectoriesAPI{
        store: store,
    }
}

// RegisterRoutes 注册目录相关路由
func (d *DirectoriesAPI) RegisterRoutes(router *mux.Router) {
    router.HandleFunc("/dirs/{path:.*}", d.ListDirectory).Methods("GET")
    router.HandleFunc("/dirs/{path:.*}", d.CreateDirectory).Methods("POST")
    router.HandleFunc("/dirs/{path:.*}", d.DeleteDirectory).Methods("DELETE")
}

// ListDirectory 列出目录内容
func (d *DirectoriesAPI) ListDirectory(w http.ResponseWriter, r *http.Request) {
    dirPath := api.ExtractPath(r)
    if dirPath == "" {
        api.RespondError(w, r, http.StatusBadRequest, 
            errors.New(errors.InvalidArgument, "无效的目录路径"))
        return
    }

    // 使用工具函数处理recursive参数
    recursive, err := utils.ParseBoolParam(r, "recursive", false)
    if err != nil {
        api.RespondError(w, r, http.StatusBadRequest, err)
        return
    }
    
    // 使用工具函数处理limit参数
    limit, err := utils.ParseIntParam(r, "limit", 100, 0, 1000)
    if err != nil {
        api.RespondError(w, r, http.StatusBadRequest, err)
        return
    }

    entries, err := d.store.ListDirectory(r.Context(), dirPath, recursive, limit)
    if err != nil {
        api.HandleAPIError(w, r, err)
        return
    }

    api.RespondSuccess(w, r, http.StatusOK, entries)
}

// CreateDirectory 创建目录
func (d *DirectoriesAPI) CreateDirectory(w http.ResponseWriter, r *http.Request) {
    dirPath := api.ExtractPath(r)
	if dirPath == "" {
		nethttp.RespondError(w, http.StatusBadRequest, "无效的目录路径")
		return
	}

    defer r.Body.Close()

	var dirInfo metadata.DirectoryInfo
	body, err := io.ReadAll(r.Body)
	if err != nil {
		// 处理错误
		api.RespondError(w, r, http.StatusBadRequest, 
            errors.New(errors.Internal, "读取请求体失败"))
		return
	}
	
	// 尝试解析请求体，但允许为空
	if len(body) > 0 {
		if err := json.Unmarshal(body, &dirInfo); err != nil {
			api.RespondError(w, r, http.StatusBadRequest, 
                errors.New(errors.InvalidArgument, "无效的请求体"))
			return
		}
	}

	// 设置目录路径
	dirInfo.Path = dirPath

	// 创建目录
	entries, err := d.store.CreateDirectory(r.Context(), dirInfo)
	if err != nil {
		api.HandleAPIError(w, r, err)
		return
	}

	api.RespondSuccess(w, r, http.StatusOK, entries)
}

// DeleteDirectory 删除目录
func (d *DirectoriesAPI) DeleteDirectory(w http.ResponseWriter, r *http.Request) {
    dirPath := api.ExtractPath(r)
    if dirPath == "" {
        api.RespondError(w, r, http.StatusBadRequest, 
            errors.New(errors.InvalidArgument, "无效的目录路径"))
        return
    }

    // 使用工具函数处理recursive参数
    recursive, err := utils.ParseBoolParam(r, "recursive", false)
    if err != nil {
        api.RespondError(w, r, http.StatusBadRequest, err)
        return
    }

    err = d.store.DeleteDirectory(r.Context(), dirPath, recursive)
    if err != nil {
        api.HandleAPIError(w, r, err)
        return
    }

    api.RespondSuccess(w, r, http.StatusOK, nil)
}