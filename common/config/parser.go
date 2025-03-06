package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	tomlv2 "github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// Parser 定义配置解析器接口
type Parser interface {
	Parse(data []byte, config interface{}) error
	ParseFile(filePath string, config interface{}) error
}

// DefaultParser 提供默认的配置解析器实现
type DefaultParser struct{}

// NewParser 创建一个新的配置解析器
func NewParser() Parser {
	return &DefaultParser{}
}

// Parse 根据数据内容解析配置
func (p *DefaultParser) Parse(data []byte, config interface{}) error {
	return json.Unmarshal(data, config)
}

// ParseFile 从文件中读取并解析配置
func (p *DefaultParser) ParseFile(filePath string, config interface{}) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		return json.Unmarshal(data, config)
	case ".yaml", ".yml":
		return yaml.Unmarshal(data, config)
	case ".toml":
		_, err := toml.Decode(string(data), config)
		return err
	default:
		return fmt.Errorf("不支持的配置文件格式: %s", ext)
	}
}

// LoadConfig 加载YAML配置
func LoadConfig(path string) (*SystemConfig, error) {
	// 默认配置
	config := &SystemConfig{}

	// 应用结构体标签中的默认值
	applyDefaults(reflect.ValueOf(config))

	// 尝试加载配置文件
	if _, err := os.Stat(path); err == nil {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("无法打开配置文件: %v", err)
		}
		defer file.Close()

		decoder := yaml.NewDecoder(file)
		if err := decoder.Decode(config); err != nil {
			return nil, fmt.Errorf("YAML解析失败: %v", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("检查配置文件状态出错: %v", err)
	} // 如果文件不存在，使用默认值继续

	// 加载环境变量并覆盖配置项
	if err := loadEnvVars(reflect.ValueOf(config)); err != nil {
		return nil, err
	}

	// 校验配置
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// LoadConfigJSON 从JSON文件加载配置
func LoadConfigJSON(path string) (*SystemConfig, error) {
	// 创建默认配置
	config := &SystemConfig{}

	// 应用结构体标签中的默认值
	applyDefaults(reflect.ValueOf(config))

	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("无法读取配置文件: %w", err)
		}

		// 直接解析到 config
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("JSON解析失败: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("检查配置文件状态出错: %w", err)
	}

	// 加载环境变量并覆盖配置项
	if err := loadEnvVars(reflect.ValueOf(config)); err != nil {
		return nil, err
	}

	// 验证配置
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// LoadConfigTOML 从TOML文件加载配置
func LoadConfigTOML(path string) (*SystemConfig, error) {
	// 创建默认配置
	config := &SystemConfig{}

	// 应用结构体标签中的默认值
	applyDefaults(reflect.ValueOf(config))

	// 尝试加载配置文件
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("无法读取配置文件: %w", err)
		}

		// 创建一个临时结构体来正确处理TOML格式的字段映射
		type tomlConfig SystemConfig
		tempConfig := tomlConfig{}

		if err := tomlv2.Unmarshal(data, &tempConfig); err != nil {
			return nil, fmt.Errorf("TOML解析失败: %w", err)
		}

		// 将临时结构体转换回SystemConfig
		*config = SystemConfig(tempConfig)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("检查配置文件状态出错: %w", err)
	}

	// 加载环境变量并覆盖配置项
	if err := loadEnvVars(reflect.ValueOf(config)); err != nil {
		return nil, err
	}

	// 验证配置
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// LoadConfigAuto 根据文件扩展名自动选择解析器
func LoadConfigAuto(path string) (*SystemConfig, error) {
	ext := strings.ToLower(filepath.Ext(path))

	var cfg *SystemConfig
	var err error

	switch ext {
	case ".yaml", ".yml":
		cfg, err = LoadConfig(path)
	case ".json":
		cfg, err = LoadConfigJSON(path)
	case ".toml":
		cfg, err = LoadConfigTOML(path)
	default:
		return nil, fmt.Errorf("不支持的配置文件格式: %s", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	return cfg, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(config *SystemConfig, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}

// 环境变量处理相关
var skipEnvOverrideForTests bool = false

// loadEnvVars 加载环境变量并覆盖配置
func loadEnvVars(val reflect.Value) error {
	if skipEnvOverrideForTests {
		return nil // 测试时跳过环境变量覆盖
	}

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 处理嵌套结构体
		if field.Kind() == reflect.Struct {
			if err := loadEnvVars(field); err != nil {
				return err
			}
			continue
		}

		envTag := fieldType.Tag.Get("env")
		if envTag != "" {
			envValue := os.Getenv(envTag)
			if envValue != "" {
				// 根据字段类型转换环境变量值
				switch field.Kind() {
				case reflect.String:
					field.SetString(envValue)
				case reflect.Int:
					if intVal, err := strconv.Atoi(envValue); err == nil {
						field.SetInt(int64(intVal))
					} else {
						return fmt.Errorf("环境变量 %s 的值 %s 不是有效的整数: %v", envTag, envValue, err)
					}
				case reflect.Bool:
					if boolVal, err := strconv.ParseBool(envValue); err == nil {
						field.SetBool(boolVal)
					} else {
						return fmt.Errorf("环境变量 %s 的值 %s 不是有效的布尔值: %v", envTag, envValue, err)
					}
				}
			}
		}
	}
	return nil
}

// loadEnvVarsForConfig 环境变量加载封装
func loadEnvVarsForConfig(config *SystemConfig) error {
	return loadEnvVars(reflect.ValueOf(config))
}

// DisableEnvOverrideForTests 禁用环境变量覆盖（测试用）
func DisableEnvOverrideForTests() {
	skipEnvOverrideForTests = true
}

// EnableEnvOverrideForTests 启用环境变量覆盖（测试用）
func EnableEnvOverrideForTests() {
	skipEnvOverrideForTests = false
}

// applyDefaults 应用默认值
func applyDefaults(val reflect.Value) {
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()

	// 遍历结构体字段
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 处理嵌套结构体
		if field.Kind() == reflect.Struct {
			applyDefaults(field)
			continue
		}

		defaultTag := fieldType.Tag.Get("default")
		if defaultTag != "" && isZeroValue(field) {
			switch field.Kind() {
			case reflect.String:
				field.SetString(defaultTag)
			case reflect.Int:
				if intVal, err := strconv.Atoi(defaultTag); err == nil {
					field.SetInt(int64(intVal))
				}
			case reflect.Bool:
				if boolVal, err := strconv.ParseBool(defaultTag); err == nil {
					field.SetBool(boolVal)
				}
			}
		}
	}
}

// applyDefaultsToConfig 应用默认值封装
func applyDefaultsToConfig(config *SystemConfig) {
	applyDefaults(reflect.ValueOf(config))
}

// isZeroValue 判断值是否为零值
func isZeroValue(v reflect.Value) bool {
	return v.Interface() == reflect.Zero(v.Type()).Interface()
}
