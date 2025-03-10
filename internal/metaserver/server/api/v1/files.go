package v1

import (
    "encoding/json"
    "net/http"
    
    "github.com/22827099/DFS_v1/common/errors"
    "github.com/22827099/DFS_v1/internal/metaserver/core/metadata"
    "github.com/22827099/DFS_v1/internal/metaserver/server/api"
    "github.com/gorilla/mux"

)

// FilesAPI 处理文件相关的API请求
type FilesAPI struct {
    store metadata.Store
}

// NewFilesAPI 创建文件API处理器
func NewFilesAPI(store metadata.Store) *FilesAPI {
    return &FilesAPI{
        store: store,
    }
}

// FileRequest 文件操作请求
type FileRequest struct {
    Name     string                 `json:"name"`
    Size     int64                  `json:"size"`
    MimeType string                 `json:"mime_type"`
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// RegisterRoutes 注册文件相关路由
func (f *FilesAPI) RegisterRoutes(router *mux.Router) {
    router.HandleFunc("/files/{path:.*}", f.GetFileInfo).Methods("GET")
    router.HandleFunc("/files/{path:.*}", f.CreateFile).Methods("POST")
    router.HandleFunc("/files/{path:.*}", f.UpdateFile).Methods("PUT")
    router.HandleFunc("/files/{path:.*}", f.DeleteFile).Methods("DELETE")
}

// GetFileInfo 获取文件信息
func (f *FilesAPI) GetFileInfo(w http.ResponseWriter, r *http.Request) {
    filePath := api.ExtractPath(r)
    if filePath == "" {
		api.RespondError(w, r, http.StatusBadRequest, 
			errors.New(errors.InvalidArgument, "无效的文件路径"))
        return
    }

    fileInfo, err := f.store.GetFileInfo(r.Context(), filePath)
    if err != nil {
        api.HandleAPIError(w, r, err)
        return
    }

    api.RespondSuccess(w, r, http.StatusOK, fileInfo)
}

// CreateFile 创建文件
func (f *FilesAPI) CreateFile(w http.ResponseWriter, r *http.Request) {
    filePath := api.ExtractPath(r)
    if filePath == "" {
        api.RespondError(w, r, http.StatusBadRequest, 
            errors.New(errors.InvalidArgument, "无效的文件路径"))
        return
    }

    // 验证请求体大小
    if r.ContentLength > 1024*1024 {
        api.RespondError(w, r, http.StatusRequestEntityTooLarge, 
            errors.New(errors.ResourceExhausted, "请求体过大"))
        return
    }

    var fileReq FileRequest
    if err := json.NewDecoder(r.Body).Decode(&fileReq); err != nil {
        api.RespondError(w, r, http.StatusBadRequest, 
            errors.New(errors.InvalidArgument, "无效的请求体: %v", err))
        return
    }
    defer r.Body.Close()

    // 验证必填字段
    if fileReq.Size < 0 {
        api.RespondError(w, r, http.StatusBadRequest, 
            errors.New(errors.InvalidArgument, "文件大小不能为负"))
        return
    }

    // 转换为存储模型
    fileInfo := metadata.FileInfo{
        Path:     filePath,
        Size:     fileReq.Size,
        MimeType: fileReq.MimeType,
        // 其他字段设置...
    }

    // 创建文件元数据
    result, err := f.store.CreateFile(r.Context(), fileInfo)
    if err != nil {
        api.HandleAPIError(w, r, err)
        return
    }

    api.RespondSuccess(w, r, http.StatusCreated, result)
}

// UpdateFile 更新文件信息
func (s *FilesAPI) UpdateFile(w http.ResponseWriter, r *http.Request) {
	filePath := api.ExtractPath(r)
	if filePath == "" {
		api.RespondError(w, r, http.StatusBadRequest, 
			errors.New(errors.InvalidArgument, "无效的文件路径"))
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		api.RespondError(w, r, http.StatusBadRequest,
			errors.New(errors.InvalidArgument, "无效的请求体"))
		return
	}
	defer r.Body.Close()

	// 更新文件元数据
	result, err := s.store.UpdateFile(r.Context(), filePath, updates)
	if err != nil {
		api.HandleAPIError(w, r, err)
		return
	}

	api.RespondSuccess(w, r, http.StatusOK, result)
}

// DeleteFile 删除文件
func (s *FilesAPI) DeleteFile(w http.ResponseWriter, r *http.Request) {
	filePath := api.ExtractPath(r)
	if filePath == "" {
		api.RespondError(w, r, http.StatusBadRequest, 
			errors.New(errors.InvalidArgument, "无效的文件路径"))
		return
	}

	err := s.store.DeleteFile(r.Context(), filePath)
	if err != nil {
        api.HandleAPIError(w, r, err)
		return
	}

    api.RespondSuccess(w, r, http.StatusOK, nil)
}