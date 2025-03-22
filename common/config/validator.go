package config

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/22827099/DFS_v1/common/config/internal/reflection"
)

// Validator 配置验证器接口
type Validator interface {
	Validate(config interface{}) error
}

// DefaultValidator 默认配置验证器实现
type DefaultValidator struct {
	rules map[string]ValidationRule
}

// ValidationRule 定义一个配置验证规则
type ValidationRule struct {
	Required  bool
	Validator func(interface{}) error
}

// NewValidator 创建一个配置验证器
func NewValidator() *DefaultValidator {
	return &DefaultValidator{
		rules: make(map[string]ValidationRule),
	}
}

// AddRule 添加验证规则
func (v *DefaultValidator) AddRule(field string, required bool, validator func(interface{}) error) {
	v.rules[field] = ValidationRule{
		Required:  required,
		Validator: validator,
	}
}

// Validate 验证配置对象
func (v *DefaultValidator) Validate(config interface{}) error {
	val := reflect.ValueOf(config)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return errors.New("配置必须是一个结构体")
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldName := field.Name

		rule, exists := v.rules[fieldName]
		if !exists {
			continue
		}

		fieldVal := val.Field(i)  // 保存为 reflect.Value

		// 检查必填字段
		if rule.Required {	
			if reflection.IsZeroValue(val.Field(i)) {
				return fmt.Errorf("字段 %s 为必填项", fieldName)
			}
		}

		// 应用自定义验证规则
		if rule.Validator != nil {
			if err := rule.Validator(fieldVal.Interface()); err != nil {
				return fmt.Errorf("字段 %s 验证失败: %w", fieldName, err)
			}
		}
	}

	return nil
}

// configEquals 比较两个配置是否相等
func configEquals(a, b *SystemConfig) bool {
	if a == nil || b == nil {
		return a == b
	}

	// 比较各个字段
	if a.NodeID != b.NodeID ||
		a.MetaServer != b.MetaServer ||
		a.DataDir != b.DataDir ||
		a.ChunkSize != b.ChunkSize ||
		a.Replicas != b.Replicas ||
		a.Logging.Level != b.Logging.Level ||
		a.Logging.Console != b.Logging.Console ||
		a.Logging.File != b.Logging.File {
		return false
	}

	return true
}
