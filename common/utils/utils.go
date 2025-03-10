package utils

import (
	"net/http"
	"strconv"

	"github.com/22827099/DFS_v1/common/errors"
)	

// parseBoolParam 解析布尔型查询参数
func ParseBoolParam(r *http.Request, name string, defaultValue bool) (bool, error) {
    param := r.URL.Query().Get(name)
    if param == "" {
        return defaultValue, nil
    }
    
    value, err := strconv.ParseBool(param)
    if err != nil {
        return defaultValue, errors.New(errors.InvalidArgument, 
            "%s参数必须是布尔值", name)
    }
    
    return value, nil
}

// parseIntParam 解析整数型查询参数
func ParseIntParam(r *http.Request, name string, defaultValue, minValue, maxValue int) (int, error) {
    param := r.URL.Query().Get(name)
    if param == "" {
        return defaultValue, nil
    }
    
    value, err := strconv.Atoi(param)
    if err != nil {
        return defaultValue, errors.New(errors.InvalidArgument, 
            "%s参数必须是整数", name)
    }
    
    if value < minValue {
        return defaultValue, errors.New(errors.InvalidArgument, 
            "%s参数不能小于%d", name, minValue)
    }
    
    if maxValue > 0 && value > maxValue {
        return maxValue, nil // 自动截断到最大值
    }
    
    return value, nil
}