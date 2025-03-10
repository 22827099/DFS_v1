package api

import (
	"encoding/json"
	"net/http"
    "path"
    "strings"

	"github.com/22827099/DFS_v1/common/errors"
	nethttp "github.com/22827099/DFS_v1/common/network/http"
	"github.com/gorilla/mux"
)

// ResponseStatus 表示API响应状态
type ResponseStatus string

const (
    // StatusSuccess 成功状态
    StatusSuccess ResponseStatus = "success"
    // StatusError 错误状态
    StatusError ResponseStatus = "error"
)

// Response 统一API响应格式
type Response struct {
    Status  ResponseStatus  `json:"status"`
    Data    interface{}     `json:"data,omitempty"`
    Error   *ErrorInfo      `json:"error,omitempty"`
    TraceID string          `json:"trace_id,omitempty"` // 用于请求追踪
}

// ErrorInfo 详细错误信息
type ErrorInfo struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

// RespondSuccess 返回成功响应
func RespondSuccess(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
    resp := Response{
        Status:  StatusSuccess,
        Data:    data,
        TraceID: nethttp.GetRequestID(r.Context()),
    }
    
    nethttp.RespondJSON(w, code, resp)
}

// RespondError 返回错误响应
func RespondError(w http.ResponseWriter, r *http.Request, code int, err error) {
    var errInfo *ErrorInfo
    
    // 如果是系统错误类型，映射错误码
    if e, ok := err.(*errors.Error); ok {
        errInfo = &ErrorInfo{
            Code:    mapErrorCode(e.Code),
            Message: e.Message,
        }
        
        // 如果有元数据，添加为详情
        if e.Metadata != nil {
            if details, err := json.Marshal(e.Metadata); err == nil {
                errInfo.Details = string(details)
            }
        }
    } else {
        // 普通错误类型
        errInfo = &ErrorInfo{
            Code:    "internal_error",
            Message: err.Error(),
        }
    }
    
    resp := Response{
        Status:  StatusError,
        Error:   errInfo,
        TraceID: nethttp.GetRequestID(r.Context()),
    }
    
    nethttp.RespondJSON(w, code, resp)
}

// 映射内部错误码到API错误码
func mapErrorCode(code errors.ErrorCode) string {
    switch code {
    case errors.NotFound:
        return "resource_not_found"
    case errors.InvalidArgument:
        return "invalid_argument"
    case errors.PermissionDenied:
        return "permission_denied"
    case errors.AlreadyExists:
        return "resource_already_exists"
    case errors.ResourceExhausted:
        return "resource_exhausted"
    case errors.Internal:
        return "internal_server_error"
    default:
        return "internal_error"
    }
}

// HandleAPIError 处理API错误并返回适当的HTTP响应
func HandleAPIError(w http.ResponseWriter, r *http.Request, err error) {
    // 根据错误类型确定状态码
    statusCode := http.StatusInternalServerError
    
    if errors.IsNotFound(err) {
        statusCode = http.StatusNotFound
    } else if errors.IsInvalidArgument(err) {
        statusCode = http.StatusBadRequest
    } else if errors.IsPermissionDenied(err) {
        statusCode = http.StatusForbidden
    } else if errors.IsAlreadyExists(err) {
        statusCode = http.StatusConflict
    } else if errors.IsUnauthenticated(err) {
        statusCode = http.StatusUnauthorized
    } else if errors.IsResourceExhausted(err) {
        statusCode = http.StatusRequestEntityTooLarge // 413 Payload Too Large
    } else if errors.IsInternal(err) {
        statusCode = http.StatusInternalServerError
    }
    
    // 返回标准错误响应
    RespondError(w, r, statusCode, err)
}

// extractPath 从请求中提取文件或目录的路径
func ExtractPath(r *http.Request) string {
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