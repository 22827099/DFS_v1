package middleware

import (
    "net/http"
    "context"
    "log"
    
    nethttp "github.com/22827099/DFS_v1/common/network/http"
)

// TransactionManager 事务管理器接口
type TransactionManager interface {
    Begin(ctx context.Context) (string, error)
    Commit(ctx context.Context, txID string) error
    Rollback(ctx context.Context, txID string) error
}

// contextKey 定义上下文键类型
type contextKey string

// 事务ID上下文键
const txIDKey contextKey = "transaction-id"

// Transaction 创建事务中间件
func Transaction(txManager TransactionManager) nethttp.Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 仅对写操作启用事务
            if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
                next.ServeHTTP(w, r)
                return
            }
            
            // 开始事务
            txID, err := txManager.Begin(r.Context())
            if err != nil {
                http.Error(w, "无法开始事务: "+err.Error(), http.StatusInternalServerError)
                return
            }
            
            // 创建带事务ID的上下文
            ctx := context.WithValue(r.Context(), txIDKey, txID)
            
            // 创建响应记录器
            recorder := &responseRecorder{
                ResponseWriter: w,
                statusCode:     http.StatusOK,
            }
            
            // 处理请求
            next.ServeHTTP(recorder, r.WithContext(ctx))
            
            // 根据响应状态决定提交或回滚
            if recorder.statusCode >= 200 && recorder.statusCode < 400 {
                // 成功响应，提交事务
                if err := txManager.Commit(r.Context(), txID); err != nil {
                    http.Error(w, "事务提交失败: "+err.Error(), http.StatusInternalServerError)
                } else {
                    // 失败响应，回滚事务
                    if err := txManager.Rollback(r.Context(), txID); err != nil {
                        // 记录事务回滚失败的错误
                        log.Printf("事务回滚失败: txID=%s, error=%v", txID, err)
                    }
                }
            }
        })
    }
}

// GetTransactionID 从上下文获取事务ID
func GetTransactionID(ctx context.Context) (string, bool) {
    txID, ok := ctx.Value(txIDKey).(string)
    return txID, ok
}

// 响应记录器，用于捕获响应状态码
type responseRecorder struct {
    http.ResponseWriter
    statusCode int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
    r.statusCode = statusCode
    r.ResponseWriter.WriteHeader(statusCode)
}