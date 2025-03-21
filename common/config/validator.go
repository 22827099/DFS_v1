package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/go-playground/validator/v10"
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

		fieldValue := val.Field(i).Interface()

		// 检查必填字段
		if rule.Required {
			if isZero(fieldValue) {
				return fmt.Errorf("字段 %s 为必填项", fieldName)
			}
		}

		// 应用自定义验证规则
		if rule.Validator != nil {
			if err := rule.Validator(fieldValue); err != nil {
				return fmt.Errorf("字段 %s 验证失败: %w", fieldName, err)
			}
		}
	}

	return nil
}

// isZero 检查值是否为零值
func isZero(v interface{}) bool {
	return reflect.DeepEqual(v, reflect.Zero(reflect.TypeOf(v)).Interface())
}

// ValidateConfig 增强的配置验证
func ValidateConfig(config *SystemConfig) error {
	validate := validator.New()

	// 注册自定义验证函数
	validate.RegisterValidation("path_exists", validatePathExists)

	// 执行验证
	if err := validate.Struct(config); err != nil {
		return fmt.Errorf("配置校验失败: %v", err)
	}

	// 在config包的ValidateConfig函数中添加
	if config.NodeID == "" {
		return fmt.Errorf("节点ID不能为空")
	}

	// 执行自定义业务规则验证
	if config.ChunkSize < 512 {
		return fmt.Errorf("块大小不能小于512字节")
	}

	if config.Replicas < 1 {
		return fmt.Errorf("副本数不能小于1")
	}

	return nil
}

// validatePathExists 验证路径是否存在
func validatePathExists(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
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
