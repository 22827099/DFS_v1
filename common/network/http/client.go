package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
    "bytes"
    "fmt"

	"github.com/22827099/DFS_v1/common/errors"
)

// Client 是HTTP客户端的简单封装
type Client struct {
	baseURL    string
	httpClient *http.Client
    headers     map[string]string
    retryPolicy *RetryPolicy  // 添加重试策略字段
}

// NewClient 创建新的HTTP客户端
func NewClient(baseURL string, options ...ClientOption) *Client {
    client := &Client{
        httpClient:  &http.Client{},
        baseURL:     baseURL,
        headers:     make(map[string]string),
        retryPolicy: DefaultRetryPolicy(),  // 设置默认重试策略
    }
    
    // 应用配置选项
    for _, option := range options {
        option(client)
    }
    
    return client
}

// Get 发送GET请求
func (c *Client) Get(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	// 添加请求头
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.httpClient.Do(req)
}

// Post 发送POST请求
func (c *Client) Post(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.httpClient.Do(req)
}

// Put 发送PUT请求
func (c *Client) Put(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+path, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.httpClient.Do(req)
}

// Delete 发送DELETE请求
func (c *Client) Delete(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	var bodyReader io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = strings.NewReader(string(jsonBody))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.httpClient.Do(req)
}

// GetJSON 发送GET请求并解析JSON响应
func (c *Client) GetJSON(ctx context.Context, path string, result interface{}) error {
	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return errors.New(errors.NetworkError, "HTTP请求失败: %d %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// PostJSON 发送POST请求并解析JSON响应到result
func (c *Client) PostJSON(ctx context.Context, path string, body interface{}, result interface{}, headers map[string]string) error {
    // 将请求体序列化为JSON
    var reqBody io.Reader
    if body != nil {
        jsonData, err := json.Marshal(body)
        if err != nil {
            return fmt.Errorf("序列化JSON失败: %w", err)
        }
        reqBody = bytes.NewBuffer(jsonData)
    }
    
    // 设置请求头
    if headers == nil {
        headers = make(map[string]string)
    }
    // 确保设置了 Content-Type
    if _, exists := headers["Content-Type"]; !exists {
        headers["Content-Type"] = "application/json"
    }
    
    // 发送POST请求
    resp, err := c.Post(ctx, path, reqBody, headers)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // 检查响应状态码
    if resp.StatusCode >= 400 {
        return fmt.Errorf("HTTP错误: %d %s", resp.StatusCode, resp.Status)
    }
    
    // 解析JSON响应到result
    if result != nil {
        if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
            return fmt.Errorf("解析响应失败: %w", err)
        }
    }
    
    return nil
}

// PutJSON 发送PUT请求并解析JSON响应
func (c *Client) PutJSON(ctx context.Context, path string, body interface{}, result interface{}, headers map[string]string) error {
	resp, err := c.Put(ctx, path, body, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return errors.New(errors.NetworkError, "HTTP请求失败: %d %s", resp.StatusCode, string(bodyBytes))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// DeleteJSON 发送DELETE请求并解析JSON响应
func (c *Client) DeleteJSON(ctx context.Context, path string, body interface{}, result interface{}, headers map[string]string) error {
	resp, err := c.Delete(ctx, path, body, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return errors.New(errors.NetworkError, "HTTP请求失败: %d %s", resp.StatusCode, string(bodyBytes))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// WithTimeout 设置客户端超时时间
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithRetryPolicy 设置重试策略
func WithRetryPolicy(policy *RetryPolicy) ClientOption {
	return func(c *Client) {
		c.retryPolicy = policy
	}
}

// DefaultRetryPolicy 返回默认的重试策略
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    3,
		RetryInterval: 500 * time.Millisecond,
		MaxBackoff:    5 * time.Second,
		ShouldRetry: func(resp *http.Response, err error) bool {
			return err != nil || (resp != nil && resp.StatusCode >= 500)
		},
	}
}

// RetryPolicy 定义HTTP请求重试策略
type RetryPolicy struct {
	MaxRetries    int
	RetryInterval time.Duration
	MaxBackoff    time.Duration
	ShouldRetry   func(*http.Response, error) bool
}

// DoWithRetry 发送HTTP请求并根据重试策略重试
func (c *Client) DoWithRetry(req *http.Request) (*http.Response, error) {
    var resp *http.Response
    var err error
    var retryCount int
    
    for retryCount = 0; retryCount <= c.retryPolicy.MaxRetries; retryCount++ {
        // 如果不是第一次尝试，则等待
        if retryCount > 0 {
            // 计算退避时间，可以使用指数退避算法
            backoffTime := c.retryPolicy.RetryInterval * time.Duration(1<<uint(retryCount-1))
            if backoffTime > c.retryPolicy.MaxBackoff {
                backoffTime = c.retryPolicy.MaxBackoff
            }
            time.Sleep(backoffTime)
            
            // 创建新的请求体（如果原始请求有请求体）
            if req.Body != nil {
                // 这里假设请求体可以被重复读取
                // 实际使用时可能需要克隆请求体
            }
        }
        
        // 发送请求
        resp, err = c.httpClient.Do(req)
        
        // 检查是否需要重试
        if !c.retryPolicy.ShouldRetry(resp, err) {
            break
        }
    }
    
    return resp, err
}

// ClientOption 定义客户端选项函数
type ClientOption func(*Client)
