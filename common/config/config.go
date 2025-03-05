package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// SystemConfig 配置结构体
type SystemConfig struct {
	NodeID     string `yaml:"node_id" env:"NODE_ID" required:"true"`
	MetaServer string `yaml:"meta_server" env:"META_ADDR" default:"localhost:8080"`
	DataDir    string `yaml:"data_dir" env:"DATA_DIR" default:"./data"`
	ChunkSize  int    `yaml:"chunk_size" env:"CHUNK_SIZE" default:"1024"` // 环境变量和默认值
	Replicas   int    `yaml:"replicas" env:"REPLICAS" default:"2"`        // 环境变量和默认值
}

// LoadConfig 加载配置
func LoadConfig(path string) (*SystemConfig, error) {
	// 1. 从文件加载YAML配置
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("无法打开配置文件: %v", err)
	}
	defer file.Close()

	var config SystemConfig
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("YAML解析失败: %v", err)
	}

	// 2. 加载环境变量并覆盖配置项
	if err := loadEnvVars(&config); err != nil {
		return nil, err
	}

	// 3. 校验配置
	validate := validator.New()
	if err := validate.Struct(&config); err != nil {
		return nil, fmt.Errorf("配置校验失败: %v", err)
	}

	return &config, nil
}

// loadEnvVars 从环境变量加载并覆盖配置
func loadEnvVars(config *SystemConfig) error {
	val := reflect.ValueOf(config).Elem()

	// 遍历结构体字段并检查环境变量
	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		envTag := field.Tag.Get("env")
		if envTag != "" {
			envValue := os.Getenv(envTag)
			if envValue != "" {
				switch field.Type.Kind() {
				case reflect.String:
					val.Field(i).SetString(envValue)
				case reflect.Int:
					intValue, err := strconv.Atoi(envValue)
					if err != nil {
						return fmt.Errorf("环境变量 %s 转换为整数时出错: %v", envTag, err)
					}
					val.Field(i).SetInt(int64(intValue))
				}
			}
		}
	}

	return nil
}
