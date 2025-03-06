package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// LoggingConfig 日志配置结构体
type LoggingConfig struct {
	Level   string `yaml:"level" json:"level" toml:"level" env:"LOG_LEVEL" default:"info"`
	Console bool   `yaml:"console" json:"console" toml:"console" env:"LOG_CONSOLE" default:"true"`
	File    string `yaml:"file" json:"file" toml:"file" env:"LOG_FILE" default:"logs/app.log"`
}

// SystemConfig 配置结构体
type SystemConfig struct {
	NodeID     string        `yaml:"node_id" json:"node_id" toml:"node_id" env:"NODE_ID" required:"true"`
	MetaServer string        `yaml:"meta_server" json:"meta_server" toml:"meta_server" env:"META_ADDR" default:"localhost:8080"`
	DataDir    string        `yaml:"data_dir" json:"data_dir" toml:"data_dir" env:"DATA_DIR" default:"./data"`
	ChunkSize  int           `yaml:"chunk_size" json:"chunk_size" toml:"chunk_size" env:"CHUNK_SIZE" default:"1024"`
	Replicas   int           `yaml:"replicas" json:"replicas" toml:"replicas" env:"REPLICAS" default:"2"`
	Logging    LoggingConfig `yaml:"logging" json:"logging" toml:"logging"`
}

// LoadConfig 加载配置
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

	// 2. 加载环境变量并覆盖配置项
	if err := loadEnvVars(reflect.ValueOf(config)); err != nil {
		return nil, err
	}

	// 3. 校验配置
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

		// 直接解析到 config，不通过临时结构体转换
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

		if err := toml.Unmarshal(data, &tempConfig); err != nil {
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

// 添加到 config.go 中
var skipEnvOverrideForTests bool = false

// 修改 loadEnvVars 函数
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

// 添加帮助函数供测试使用
func DisableEnvOverrideForTests() {
	skipEnvOverrideForTests = true
}

func EnableEnvOverrideForTests() {
	skipEnvOverrideForTests = false
}

// 修改调用方式
func loadEnvVarsForConfig(config *SystemConfig) error {
	return loadEnvVars(reflect.ValueOf(config))
}

// 添加设置默认值的函数
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

// 修改调用方式
func applyDefaultsToConfig(config *SystemConfig) {
	applyDefaults(reflect.ValueOf(config))
}

// 判断值是否为零值
func isZeroValue(v reflect.Value) bool {
	return v.Interface() == reflect.Zero(v.Type()).Interface()
}

// ConfigWatcher 配置监视器
type ConfigWatcher struct {
	ConfigPath string
	Config     *SystemConfig
	onChange   func(*SystemConfig)
	stopChan   chan struct{}
}

// NewConfigWatcher 创建新的配置监视器
func NewConfigWatcher(path string, onChange func(*SystemConfig)) (*ConfigWatcher, error) {
	watcher := &ConfigWatcher{
		ConfigPath: path,
		onChange:   onChange,
		stopChan:   make(chan struct{}),
	}

	// 初始加载配置
	config, err := LoadConfigAuto(path)
	if err != nil {
		return nil, err
	}

	watcher.Config = config

	return watcher, nil
}

// Start 开始监视配置文件变化
func (w *ConfigWatcher) Start() {
	go func() {
		ticker := time.NewTicker(30 * time.Second) // 定期检查
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.checkAndReload()
			case <-w.stopChan:
				return
			}
		}
	}()
}

// Stop 停止监视
func (w *ConfigWatcher) Stop() {
	close(w.stopChan)
}

// checkAndReload 检查并重新加载配置
func (w *ConfigWatcher) checkAndReload() {
	if w.ConfigPath == "" {
        return // 避免路径为空
    }

	// 使用 LoadConfigAuto 代替手动分派
	config, err := LoadConfigAuto(w.ConfigPath)
	if err != nil {
		fmt.Printf("重新加载配置失败: %v\n", err)
		return
	}

	// 使用正确的相等比较
	if !configEquals(w.Config, config) {
		w.Config = config
		if w.onChange != nil {
			w.onChange(config)
		}
	}
}

// 添加辅助函数比较两个配置是否相等
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

// ValidateConfig 增强的配置验证
func ValidateConfig(config *SystemConfig) error {
	validate := validator.New()

	// 注册自定义验证函数
	validate.RegisterValidation("path_exists", validatePathExists)

	// 执行验证
	if err := validate.Struct(config); err != nil {
		return fmt.Errorf("配置校验失败: %v", err)
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

// 自定义验证函数
func validatePathExists(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
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
