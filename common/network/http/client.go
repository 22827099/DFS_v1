package http

import (
    "context"
    "encoding/json"
    "io"
    "net/http"
    "strings"
    "time"

    "github.com/22827099/DFS_v1/common/errors"
)

// Client 是HTTP客户端的简单封装
type Client struct {
    baseURL    string
    httpClient *http.Client
}

// NewClient 创建一个新的HTTP客户端
func NewClient(baseURL string) *Client {
    return &Client{
        baseURL: baseURL,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
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

// PostJSON 发送POST请求并解析JSON响应
func (c *Client) PostJSON(ctx context.Context, path string, body interface{}, result interface{}) error {
    resp, err := c.Post(ctx, path, body, nil)
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