package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/22827099/DFS_v1/common/errors"
	"github.com/22827099/DFS_v1/common/logging"
)

// Client 是HTTP客户端封装
type Client struct {
	client      *http.Client
	baseURL     string
	headers     map[string]string
	retryPolicy *RetryPolicy
	logger      Logger
}

// DefaultRetryPolicy 返回默认重试策略
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    3,
		RetryInterval: 1 * time.Second,
		MaxBackoff:    30 * time.Second,
		ShouldRetry: func(resp *http.Response, err error) bool {
			// 当遇到网络错误或服务器错误(5xx)时重试
			return err != nil || (resp != nil && resp.StatusCode >= 500)
		},
	}
}

// NewClient 创建新的HTTP客户端
func NewClient(baseURL string, options ...ClientOption) *Client {
	client := &Client{
		client: &http.Client{
			Timeout: 30 * time.Second, // 默认超时时间
		},
		baseURL: baseURL,
		headers: make(map[string]string),
		logger:  logging.NewLogger(), // 使用系统默认日志记录器
	}

	// 应用选项
	for _, option := range options {
		option(client)
	}

	return client
}

// WithTimeout 设置客户端超时
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.client.Timeout = timeout
	}
}

// WithRetryPolicy 设置重试策略
func WithRetryPolicy(policy *RetryPolicy) ClientOption {
	return func(c *Client) {
		c.retryPolicy = policy
	}
}

// WithHeader 添加默认请求头
func WithHeader(key, value string) ClientOption {
	return func(c *Client) {
		c.headers[key] = value
	}
}

// WithLogger 设置日志记录器
func WithLogger(logger Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// request 是内部请求方法，支持重试
func (c *Client) request(ctx context.Context, method, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	// 构造URL
	requestURL, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return nil, errors.Wrap(err, errors.InvalidArgument, "无效的URL路径")
	}

	// 准备请求体
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, errors.Wrap(err, errors.InvalidArgument, "请求体JSON编码失败")
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
	if err != nil {
		return nil, errors.Wrap(err, errors.InvalidArgument, "创建HTTP请求失败")
	}

	// 设置默认请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 添加全局请求头
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// 添加特定请求头
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 如果没有重试策略，直接发送请求
	if c.retryPolicy == nil {
		return c.client.Do(req)
	}

	// 使用重试策略
	var resp *http.Response
	var lastErr error
	retries := 0

	for retries <= c.retryPolicy.MaxRetries {
		if retries > 0 {
			// 确认请求上下文是否已取消
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				// 继续重试
			}

			// 计算退避时间
			backoff := c.calculateBackoff(retries)
			c.logger.Info("HTTP请求重试中，尝试次数: %d, 等待时间: %v", retries, backoff)

			// 等待退避时间
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
				// 继续重试
			}
		}

		// 发送请求
		resp, lastErr = c.client.Do(req)

		// 检查是否需要重试
		if !c.retryPolicy.ShouldRetry(resp, lastErr) {
			break
		}

		// 如果有响应但需要重试，先关闭body
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}

		retries++
	}

	if lastErr != nil {
		return nil, errors.Wrap(lastErr, errors.NetworkError, "HTTP请求失败，已重试%d次", retries)
	}

	return resp, nil
}

// calculateBackoff 计算指数退避时间
func (c *Client) calculateBackoff(retries int) time.Duration {
	// 指数退避算法：baseInterval * 2^retries + jitter
	backoff := float64(c.retryPolicy.RetryInterval) * math.Pow(2, float64(retries))

	// 添加一些随机性，避免同时重试
	jitter := rand.Float64() * float64(c.retryPolicy.RetryInterval)
	backoff += jitter

	// 确保不超过最大退避时间
	if backoff > float64(c.retryPolicy.MaxBackoff) {
		backoff = float64(c.retryPolicy.MaxBackoff)
	}

	return time.Duration(backoff)
}

// Get 发送GET请求
func (c *Client) Get(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	return c.request(ctx, http.MethodGet, path, nil, headers)
}

// Post 发送POST请求
func (c *Client) Post(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	return c.request(ctx, http.MethodPost, path, body, headers)
}

// Put 发送PUT请求
func (c *Client) Put(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	return c.request(ctx, http.MethodPut, path, body, headers)
}

// Delete 发送DELETE请求
func (c *Client) Delete(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	return c.request(ctx, http.MethodDelete, path, nil, headers)
}

// GetJSON 发送GET请求并将响应解析为JSON
func (c *Client) GetJSON(ctx context.Context, path string, headers map[string]string, result interface{}) error {
	resp, err := c.Get(ctx, path, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return handleJSONResponse(resp, result)
}

// PostJSON 发送POST请求并将响应解析为JSON
func (c *Client) PostJSON(ctx context.Context, path string, body, result interface{}, headers map[string]string) error {
	resp, err := c.Post(ctx, path, body, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return handleJSONResponse(resp, result)
}

// PutJSON 发送PUT请求并将响应解析为JSON
func (c *Client) PutJSON(ctx context.Context, path string, body, result interface{}, headers map[string]string) error {
	resp, err := c.Put(ctx, path, body, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return handleJSONResponse(resp, result)
}

// DeleteJSON 发送DELETE请求并将响应解析为JSON
func (c *Client) DeleteJSON(ctx context.Context, path string, result interface{}, headers map[string]string) error {
	resp, err := c.Delete(ctx, path, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return handleJSONResponse(resp, result)
}

// handleJSONResponse 处理JSON响应
func handleJSONResponse(resp *http.Response, result interface{}) error {
	// 检查HTTP状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return errors.New(errors.NetworkError, "非成功的HTTP状态: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析JSON响应
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return errors.Wrap(err, errors.InvalidArgument, "解析JSON响应失败")
		}
	}

	return nil
}
