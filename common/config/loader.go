package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"

	"github.com/22827099/DFS_v1/common/config/internal/reflection"
	"github.com/22827099/DFS_v1/common/config/internal/validation"
	"github.com/22827099/DFS_v1/common/types"
)

// 统一的配置加载函数，用于统一替代原有的多个Load函数
func LoadConfig(path string, config interface{}) error {
	// 1. 应用默认值
	ApplyDefaults(config)

	// 2. 读取并解析配置文件（如果存在）
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("读取配置文件失败: %w", err)
		}

		// 根据文件扩展名解析
		ext := filepath.Ext(path)
		if err := parseConfigData(data, config, ext); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("检查配置文件状态出错: %w", err)
	}
	// 如果文件不存在，只使用默认值和环境变量

	// 3. 应用环境变量覆盖
	if err := ApplyEnvironmentVariables(config); err != nil {
		return err
	}

	// 4. 验证配置
	if err := validation.ValidateConfig(config); err != nil {
		return err
	}

	// 5. 处理特殊字段（如NodeID等）
	if err := processConfig(config); err != nil {
		return err
	}

	return nil
}

// parseConfigData 根据扩展名解析配置数据（保留原有功能）
func parseConfigData(data []byte, config interface{}, ext string) error {
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return fmt.Errorf("解析JSON配置失败: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return fmt.Errorf("解析YAML配置失败: %w", err)
		}
	case ".toml":
		if err := toml.Unmarshal(data, config); err != nil {
			return fmt.Errorf("解析TOML配置失败: %w", err)
		}
	default:
		return fmt.Errorf("不支持的配置文件格式: %s", ext)
	}
	return nil
}

// ApplyEnvironmentVariables 从环境变量中加载配置覆盖值
// 整合了parser.go中的loadEnvVars功能，增强了类型处理
func ApplyEnvironmentVariables(config interface{}) error {
	// 添加skipEnvOverrideForTests判断，保留测试功能
	if skipEnvOverrideForTests {
		return nil
	}

	val := reflect.ValueOf(config)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config必须是结构体指针")
	}

	// 获取实际的结构体值
	val = val.Elem()
	typ := val.Type()

	// 遍历所有字段
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 递归处理嵌套结构体
		if field.Kind() == reflect.Struct {
			if err := ApplyEnvironmentVariables(field.Addr().Interface()); err != nil {
				return err
			}
			continue
		}

		// 获取env标签
		envTag := fieldType.Tag.Get("env")
		if envTag == "" {
			continue
		}

		// 获取环境变量值
		envValue := os.Getenv(envTag)
		if envValue == "" {
			continue
		}

		// 设置字段值
		if err := reflection.SetFieldFromString(field, envValue); err != nil {
			return fmt.Errorf("设置字段 %s 失败: %w", fieldType.Name, err)
		}
	}

	return nil
}

// ApplyDefaults 应用默认值到配置项
// 整合了parser.go中的applyDefaults功能，增强了类型处理
func ApplyDefaults(config interface{}) {
	val := reflect.ValueOf(config)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return
	}

	// 获取实际的结构体值
	val = val.Elem()
	typ := val.Type()

	// 遍历所有字段
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 递归处理嵌套结构体
		if field.Kind() == reflect.Struct {
			ApplyDefaults(field.Addr().Interface())
			continue
		}

		// 获取default标签
		defaultTag := fieldType.Tag.Get("default")
		if defaultTag == "" {
			continue
		}

		// 如果字段是零值，则应用默认值
		if reflection.IsZeroValue(field) {
			if err := reflection.SetFieldFromString(field, defaultTag); err != nil {
				// Just log or ignore the error for default values
				continue
			}
		}
	}
}

// 测试辅助变量和函数 - 从parser.go移植过来
var skipEnvOverrideForTests bool = false

// DisableEnvOverrideForTests 禁用环境变量覆盖（测试用）
func DisableEnvOverrideForTests() {
	skipEnvOverrideForTests = true
}

// EnableEnvOverrideForTests 启用环境变量覆盖（测试用）
func EnableEnvOverrideForTests() {
	skipEnvOverrideForTests = false
}

// 处理特殊配置字段 - 处理多种配置类型
func processConfig(cfg interface{}) error {
	// 支持 SystemConfig
	if sysConfig, ok := cfg.(*SystemConfig); ok {
		if sysConfig.NodeID.IsEmpty() {
			return errors.New("节点ID不能为空")
		}
		return nil
	}

	// 支持 BaseConfig（通过反射）
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		// 检查是否有 Node 字段
		nodeField := v.FieldByName("Node")
		if nodeField.IsValid() && nodeField.Kind() == reflect.Struct {
			// 检查 Node.ID 字段
			idField := nodeField.FieldByName("ID")
			if idField.IsValid() {
				// 检查 ID 是否为空
				if id, ok := idField.Interface().(types.NodeID); ok && id.IsEmpty() {
					return errors.New("节点ID不能为空")
				}
			}
		}
	}

	return nil
}

// LoadConfigJSON 专门用于加载JSON格式的配置文件
func LoadConfigJSON(path string) (*SystemConfig, error) {
    config := &SystemConfig{}
    if err := LoadConfig(path, config); err != nil {
        return nil, err
    }
    return config, nil
}

// LoadConfigTOML 专门用于加载TOML格式的配置文件
func LoadConfigTOML(path string) (*SystemConfig, error) {
    config := &SystemConfig{}
    if err := LoadConfig(path, config); err != nil {
        return nil, err
    }
    return config, nil
}

// LoadConfigYAML 专门用于加载YAML格式的配置文件
func LoadConfigYAML(path string) (*SystemConfig, error) {
	config := &SystemConfig{}
	if err := LoadConfig(path, config); err != nil {
		return nil, err
	}
	return config, nil
}

// LoadConfigAuto 自动检测配置文件格式并加载
func LoadConfigAuto(path string) (*SystemConfig, error) {
	// 根据文件扩展名选择加载方式
	ext := filepath.Ext(path)

	switch ext {
	case ".json":
		return LoadConfigJSON(path)
	case ".yaml", ".yml":
		return LoadSystemConfig(path)
	case ".toml":
		return LoadConfigTOML(path)
	default:
		return nil, fmt.Errorf("不支持的配置文件格式: %s", ext)
	}
}
