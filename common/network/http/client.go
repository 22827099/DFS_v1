package http

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

// Client 是HTTP客户端的简单封装
type Client struct {
    baseURL    string
    httpClient *http.Client
    retryPolicy *RetryPolicy
}

// ClientOption 定义客户端选项函数
type ClientOption func(*Client)

// RetryPolicy 定义HTTP请求重试策略
type RetryPolicy struct {
    MaxRetries    int
    RetryInterval time.Duration
    MaxBackoff    time.Duration
    ShouldRetry   func(*http.Response, error) bool
}

// NewClient 创建新的HTTP客户端
func NewClient(baseURL string, options ...ClientOption) *Client {
    client := &Client{
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        baseURL: baseURL,
        retryPolicy: &RetryPolicy{
            MaxRetries:    3,
            RetryInterval: 500 * time.Millisecond,
            MaxBackoff:    5 * time.Second,
            ShouldRetry: func(resp *http.Response, err error) bool {
                return err != nil || (resp != nil && resp.StatusCode >= 500)
            },
        },
    }
    
    for _, option := range options {
        option(client)
    }
    
    return client
}

// 基础请求方法
func (c *Client) request(ctx context.Context, method, path string, body interface{}, headers map[string]string) (*http.Response, error) {
    var bodyReader io.Reader
    
    if body != nil {
        jsonData, err := json.Marshal(body)
        if err != nil {
            return nil, fmt.Errorf("序列化请求体失败: %w", err)
        }
        bodyReader = bytes.NewReader(jsonData)
    }
    
    req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
    if err != nil {
        return nil, err
    }
    
    if body != nil {
        req.Header.Set("Content-Type", "application/json")
    }
    
    for k, v := range headers {
        req.Header.Set(k, v)
    }
    
    return c.doWithRetry(req)
}

// 带重试的请求执行
func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
    var resp *http.Response
    var err error
    
    for retryCount := 0; retryCount <= c.retryPolicy.MaxRetries; retryCount++ {
        if retryCount > 0 {
            backoffTime := c.retryPolicy.RetryInterval * time.Duration(1<<uint(retryCount-1))
            if backoffTime > c.retryPolicy.MaxBackoff {
                backoffTime = c.retryPolicy.MaxBackoff
            }
            time.Sleep(backoffTime)
            
            // 为重试创建新的请求体
            if req.Body != nil {
                req.Body.Close()
                if req.GetBody != nil {
                    newBody, err := req.GetBody()
                    if err != nil {
                        return nil, err
                    }
                    req.Body = newBody
                }
            }
        }
        
        resp, err = c.httpClient.Do(req)
        
        if !c.retryPolicy.ShouldRetry(resp, err) {
            return resp, err
        }
        
        if retryCount == c.retryPolicy.MaxRetries {
            if resp != nil {
                bodyBytes, _ := io.ReadAll(resp.Body)
                resp.Body.Close()
                return nil, fmt.Errorf("最大重试次数已达到: HTTP %d: %s", 
                    resp.StatusCode, string(bodyBytes))
            }
            return nil, fmt.Errorf("最大重试次数已达到: %w", err)
        }
        
        if resp != nil && resp.Body != nil {
            resp.Body.Close()
        }
    }
    
    return resp, err
}

// DoJSON 执行HTTP请求并处理JSON响应
func (c *Client) DoJSON(ctx context.Context, method, path string, reqBody, respBody interface{}, headers map[string]string) error {
    resp, err := c.request(ctx, method, path, reqBody, headers)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // 检查响应状态
    if resp.StatusCode >= 400 {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("HTTP请求失败: %d %s", resp.StatusCode, string(bodyBytes))
    }
    
    // 如果不需要解析响应体
    if respBody == nil {
        // 读取并丢弃响应体以允许连接复用
        _, _ = io.Copy(io.Discard, resp.Body)
        return nil
    }
    
    // 解析JSON响应
    if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
        return fmt.Errorf("解析响应失败: %w", err)
    }
    
    return nil
}

// GetJSON 发送GET请求并解析JSON响应
func (c *Client) GetJSON(ctx context.Context, path string, result interface{}) error {
    return c.DoJSON(ctx, http.MethodGet, path, nil, result, nil)
}

// PostJSON 发送POST请求并解析JSON响应
func (c *Client) PostJSON(ctx context.Context, path string, body, result interface{}) error {
    return c.DoJSON(ctx, http.MethodPost, path, body, result, nil)
}

// PutJSON 发送PUT请求并解析JSON响应
func (c *Client) PutJSON(ctx context.Context, path string, body, result interface{}) error {
    return c.DoJSON(ctx, http.MethodPut, path, body, result, nil)
}

// DeleteJSON 发送DELETE请求并解析JSON响应
func (c *Client) DeleteJSON(ctx context.Context, path string, result interface{}) error {
    return c.DoJSON(ctx, http.MethodDelete, path, nil, result, nil)
}

// WithTimeout 设置客户端超时时间
func WithClientTimeout(timeout time.Duration) ClientOption {
    return func(c *Client) {
        c.httpClient.Timeout = timeout
    }
}

// WithRetryPolicy 设置重试策略
func WithRetryPolicy(maxRetries int, retryInterval time.Duration) ClientOption {
    return func(c *Client) {
        c.retryPolicy.MaxRetries = maxRetries
        c.retryPolicy.RetryInterval = retryInterval
    }
}

// WithHTTPClient 设置自定义HTTP客户端
func WithHTTPClient(httpClient *http.Client) ClientOption {
    return func(c *Client) {
        c.httpClient = httpClient
    }
}