package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	networkHttp "github.com/22827099/DFS_v1/common/network/http"
	"github.com/gorilla/mux"
)

func TestContext_JSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	ctx := &networkHttp.Context{
		Request:  r,
		Response: w,
		Params:   make(map[string]string),
	}

	data := map[string]string{"message": "测试消息"}
	err := ctx.JSON(http.StatusOK, data)

	if err != nil {
		t.Fatalf("Context.JSON: 返回错误: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Context.JSON: 期望状态码%d，得到%d", http.StatusOK, w.Code)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Context.JSON: 期望Content-Type为'application/json'，得到'%s'", contentType)
	}

	var respData map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &respData); err != nil {
		t.Fatalf("Context.JSON: 无法解析响应: %v", err)
	}

	if respData["message"] != "测试消息" {
		t.Errorf("Context.JSON: 期望message为'测试消息'，得到'%s'", respData["message"])
	}
}

func TestContext_Text(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	ctx := &networkHttp.Context{
		Request:  r,
		Response: w,
		Params:   make(map[string]string),
	}

	ctx.Text(http.StatusOK, "测试文本")

	if w.Code != http.StatusOK {
		t.Errorf("Context.Text: 期望状态码%d，得到%d", http.StatusOK, w.Code)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "text/plain" {
		t.Errorf("Context.Text: 期望Content-Type为'text/plain'，得到'%s'", contentType)
	}

	if w.Body.String() != "测试文本" {
		t.Errorf("Context.Text: 期望响应体为'测试文本'，得到'%s'", w.Body.String())
	}
}

func TestContext_Error(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	ctx := &networkHttp.Context{
		Request:  r,
		Response: w,
		Params:   make(map[string]string),
	}

	err := ctx.Error(http.StatusBadRequest, "错误消息")

	if err != nil {
		t.Fatalf("Context.Error: 返回错误: %v", err)
	}

	var respData map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &respData); err != nil {
		t.Fatalf("Context.Error: 无法解析响应: %v", err)
	}

	if respData["error"] != "错误消息" {
		t.Errorf("Context.Error: 期望error为'错误消息'，得到'%s'", respData["error"])
	}
}

func TestContext_GetParam(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	ctx := &networkHttp.Context{
		Request:  r,
		Response: w,
		Params:   map[string]string{"id": "123"},
	}

	param := ctx.GetParam("id")
	if param != "123" {
		t.Errorf("Context.GetParam: 期望参数'id'为'123'，得到'%s'", param)
	}

	param = ctx.GetParam("unknown")
	if param != "" {
		t.Errorf("Context.GetParam: 期望参数'unknown'为空，得到'%s'", param)
	}
}

func TestContext_GetQuery(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test?name=张三&age=30", nil)

	ctx := &networkHttp.Context{
		Request:  r,
		Response: w,
		Params:   make(map[string]string),
	}

	name := ctx.GetQuery("name")
	if name != "张三" {
		t.Errorf("Context.GetQuery: 期望查询参数'name'为'张三'，得到'%s'", name)
	}

	age := ctx.GetQuery("age")
	if age != "30" {
		t.Errorf("Context.GetQuery: 期望查询参数'age'为'30'，得到'%s'", age)
	}
}

func TestContext_BindJSON(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	person := Person{Name: "张三", Age: 30}
	jsonData, err := json.Marshal(person)
	if err != nil {
		t.Fatalf("Context.BindJSON: 无法序列化测试数据: %v", err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBuffer(jsonData))
	r.Header.Set("Content-Type", "application/json")

	ctx := &networkHttp.Context{
		Request:  r,
		Response: w,
		Params:   make(map[string]string),
	}

	var result Person
	err = ctx.BindJSON(&result)
	if err != nil {
		t.Fatalf("Context.BindJSON: 返回错误: %v", err)
	}

	if result.Name != "张三" {
		t.Errorf("Context.BindJSON: 期望Name为'张三'，得到'%s'", result.Name)
	}

	if result.Age != 30 {
		t.Errorf("Context.BindJSON: 期望Age为30，得到%d", result.Age)
	}
}

func TestAdapt(t *testing.T) {
	router := mux.NewRouter()

	handler := func(c *networkHttp.Context) {
		name := c.GetParam("name")
		c.Text(http.StatusOK, "你好，"+name)
	}

	adaptedHandler := networkHttp.Adapt(handler)
	router.HandleFunc("/greet/{name}", adaptedHandler)

	r := httptest.NewRequest(http.MethodGet, "/greet/张三", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Adapt: 期望状态码%d，得到%d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "你好，张三" {
		t.Errorf("Adapt: 期望响应体为'你好，张三'，得到'%s'", w.Body.String())
	}
}
