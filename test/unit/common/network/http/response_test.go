package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	networkHttp "github.com/22827099/DFS_v1/common/network/http"
)

func TestSuccessResponse(t *testing.T) {
	// 测试成功响应的创建
	data := map[string]string{"key": "value"}
	resp := networkHttp.SuccessResponse(data)

	if !resp.Success {
		t.Errorf("SuccessResponse: 期望Success为true，得到false")
	}

	dataMap, ok := resp.Data.(map[string]string)
	if !ok {
		t.Fatalf("SuccessResponse: 期望Data类型为map[string]string，得到%T", resp.Data)
	}

	if dataMap["key"] != "value" {
		t.Errorf("SuccessResponse: 期望Data['key']为'value'，得到'%s'", dataMap["key"])
	}

	if resp.Error != nil {
		t.Errorf("SuccessResponse: 期望Error为nil，得到%v", resp.Error)
	}
}

func TestErrorResponse(t *testing.T) {
	// 不带错误码测试
	resp := networkHttp.ErrorResponse("错误消息")

	if resp.Success {
		t.Errorf("ErrorResponse: 期望Success为false，得到true")
	}

	if resp.Data != nil {
		t.Errorf("ErrorResponse: 期望Data为nil，得到%v", resp.Data)
	}

	if resp.Error == nil {
		t.Fatalf("ErrorResponse: 期望Error不为nil")
	}

	if resp.Error.Code != "" {
		t.Errorf("ErrorResponse: 期望Error.Code为空，得到'%s'", resp.Error.Code)
	}

	if resp.Error.Message != "错误消息" {
		t.Errorf("ErrorResponse: 期望Error.Message为'错误消息'，得到'%s'", resp.Error.Message)
	}

	// 带错误码测试
	resp = networkHttp.ErrorResponse("错误消息", "ERR001")

	if resp.Error.Code != "ERR001" {
		t.Errorf("ErrorResponse: 期望Error.Code为'ERR001'，得到'%s'", resp.Error.Code)
	}
}

func TestRespondJSON(t *testing.T) {
	// 测试发送普通数据为JSON
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}
	err := networkHttp.RespondJSON(w, http.StatusOK, data)

	if err != nil {
		t.Fatalf("RespondJSON: 返回错误: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("RespondJSON: 期望状态码%d，得到%d", http.StatusOK, w.Code)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("RespondJSON: 期望Content-Type为'application/json'，得到'%s'", contentType)
	}

	var resp networkHttp.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("RespondJSON: 无法解析响应: %v", err)
	}

	if !resp.Success {
		t.Errorf("RespondJSON: 期望Success为true，得到false")
	}

	// 测试发送StandardResponse
	w = httptest.NewRecorder()
	standardResp := networkHttp.ErrorResponse("错误消息")
	err = networkHttp.RespondJSON(w, http.StatusBadRequest, standardResp)

	if err != nil {
		t.Fatalf("RespondJSON: 返回错误: %v", err)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("RespondJSON: 无法解析响应: %v", err)
	}

	if resp.Success {
		t.Errorf("RespondJSON: 期望Success为false，得到true")
	}
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()
	err := networkHttp.RespondError(w, http.StatusBadRequest, "错误消息", "ERR001")

	if err != nil {
		t.Fatalf("RespondError: 返回错误: %v", err)
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("RespondError: 期望状态码%d，得到%d", http.StatusBadRequest, w.Code)
	}

	var resp networkHttp.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("RespondError: 无法解析响应: %v", err)
	}

	if resp.Error.Code != "ERR001" {
		t.Errorf("RespondError: 期望Error.Code为'ERR001'，得到'%s'", resp.Error.Code)
	}
}

func TestRespondText(t *testing.T) {
	w := httptest.NewRecorder()
	networkHttp.RespondText(w, http.StatusOK, "测试文本")

	if w.Code != http.StatusOK {
		t.Errorf("RespondText: 期望状态码%d，得到%d", http.StatusOK, w.Code)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "text/plain" {
		t.Errorf("RespondText: 期望Content-Type为'text/plain'，得到'%s'", contentType)
	}

	if w.Body.String() != "测试文本" {
		t.Errorf("RespondText: 期望响应体为'测试文本'，得到'%s'", w.Body.String())
	}
}

func TestRespondNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	networkHttp.RespondNoContent(w)

	if w.Code != http.StatusNoContent {
		t.Errorf("RespondNoContent: 期望状态码%d，得到%d", http.StatusNoContent, w.Code)
	}

	if w.Body.Len() != 0 {
		t.Errorf("RespondNoContent: 期望空响应体，得到'%s'", w.Body.String())
	}
}
