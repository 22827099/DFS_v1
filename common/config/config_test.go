package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 测试加载配置
func TestLoadConfig(t *testing.T) {
	// 设置环境变量
	setTestEnvVars()

	// 创建模拟的配置文件
	configFile := "config.yaml"
	if err := createTestConfigFile(configFile); err != nil {
		t.Fatalf("创建配置文件失败: %v", err)
	}
	defer os.Remove(configFile) // 测试结束后删除配置文件

	// 加载配置
	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("配置加载失败: %v", err)
	}

	// 验证环境变量覆盖
	assert.Equal(t, "node_1", cfg.NodeID) // 环境变量覆盖生效
	assert.Equal(t, "192.168.1.100:8080", cfg.MetaServer)
	assert.Equal(t, "/tmp/data", cfg.DataDir)
	assert.Equal(t, 2048, cfg.ChunkSize) // 环境变量覆盖YAML中的1024
	assert.Equal(t, 3, cfg.Replicas)     // 环境变量覆盖YAML中的2
}

// TestLoadConfigFileNotExists 测试不存在的配置文件情况
func TestLoadConfigFileNotExists(t *testing.T) {
	// 清除环境变量，确保使用默认值
	os.Unsetenv("META_ADDR")
	os.Unsetenv("DATA_DIR")
	os.Unsetenv("CHUNK_SIZE")
	os.Unsetenv("REPLICAS")

	// 必须设置 NODE_ID，因为它是必需的
	os.Setenv("NODE_ID", "default_node")

	// 尝试加载不存在的文件
	cfg, err := LoadConfig("non_existent_config.yaml")

	// 应该不会返回错误，而是使用默认值
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// 验证是否使用了默认值
	assert.Equal(t, "default_node", cfg.NodeID)
	assert.Equal(t, "localhost:8080", cfg.MetaServer) // 默认值
	assert.Equal(t, "./data", cfg.DataDir)            // 默认值
	assert.Equal(t, 1024, cfg.ChunkSize)              // 默认值
	assert.Equal(t, 2, cfg.Replicas)                  // 默认值
}

// TestLoadInvalidConfigFormat 测试无效配置格式
func TestLoadInvalidConfigFormat(t *testing.T) {
	// 设置必需的环境变量
	os.Setenv("NODE_ID", "test_node")

	// 创建格式错误的YAML文件
	invalidFile := "invalid_config.yaml"
	invalidContent := []byte(`
        node_id: "test_node"
        meta_server: "localhost:8080
        data_dir: "./data"
    `) // 注意这里故意少了一个引号

	err := os.WriteFile(invalidFile, invalidContent, 0644)
	assert.NoError(t, err)
	defer os.Remove(invalidFile)

	// 尝试加载无效格式的配置
	_, err = LoadConfig(invalidFile)

	// 应该返回解析错误
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "YAML解析失败")
}

// TestConfigValidationFails 测试配置验证失败
func TestConfigValidationFails(t *testing.T) {
	// 创建验证会失败的配置文件
	invalidConfigFile := "invalid_values.yaml"
	invalidContent := []byte(`
        node_id: "test_node"
        meta_server: "localhost:8080"
        data_dir: "./data"
        chunk_size: 100  # 小于最小值512
        replicas: 0      # 小于最小值1
    `)

	err := os.WriteFile(invalidConfigFile, invalidContent, 0644)
	assert.NoError(t, err)
	defer os.Remove(invalidConfigFile)

	// 尝试加载配置
	_, err = LoadConfig(invalidConfigFile)

    // 应该返回验证错误
    if assert.Error(t, err, "应该返回验证错误") {
        // 只有当err不为nil时才会执行此行
        assert.Contains(t, err.Error(), "块大小不能小于512字节") 
    }
}

// TestDefaultValuesApplied 测试默认值应用
func TestDefaultValuesApplied(t *testing.T) {
	// 清除环境变量
	os.Unsetenv("META_ADDR")
	os.Unsetenv("DATA_DIR")
	os.Unsetenv("CHUNK_SIZE")
	os.Unsetenv("REPLICAS")

	// 只提供必需的NODE_ID
	os.Setenv("NODE_ID", "minimal_node")

	// 创建最小配置文件，只包含必需字段
	minimalFile := "minimal_config.yaml"
	minimalContent := []byte(`
        node_id: "minimal_node"
    `)

	err := os.WriteFile(minimalFile, minimalContent, 0644)
	assert.NoError(t, err)
	defer os.Remove(minimalFile)

	// 加载配置
	cfg, err := LoadConfig(minimalFile)
	assert.NoError(t, err)

	// 验证默认值是否正确应用
	assert.Equal(t, "minimal_node", cfg.NodeID)
	assert.Equal(t, "localhost:8080", cfg.MetaServer) // 默认值
	assert.Equal(t, "./data", cfg.DataDir)            // 默认值
	assert.Equal(t, 1024, cfg.ChunkSize)              // 默认值
	assert.Equal(t, 2, cfg.Replicas)                  // 默认值

	// 验证日志配置的默认值
	assert.Equal(t, "info", cfg.Logging.Level)        // 默认值
	assert.Equal(t, true, cfg.Logging.Console)        // 默认值
	assert.Equal(t, "logs/app.log", cfg.Logging.File) // 默认值
}

// TestConfigHotReload 测试配置热重载
func TestConfigHotReload(t *testing.T) {
	// 方案1: 禁用环境变量覆盖
	DisableEnvOverrideForTests()
	defer EnableEnvOverrideForTests() // 测试结束后恢复
	
	// 方案2: 或者清除可能影响测试的环境变量
	os.Unsetenv("NODE_ID")

	// 创建测试配置文件
	configFile := "reload_test.yaml"
	originalContent := []byte(`
        node_id: "reload_test"
        meta_server: "localhost:8080"
        data_dir: "./data"
        chunk_size: 1024
        replicas: 2
    `)

	err := os.WriteFile(configFile, originalContent, 0644)
	assert.NoError(t, err)
	defer os.Remove(configFile)

	// 标记是否回调被调用
	callbackCalled := false
	var newConfig *SystemConfig

	// 创建配置监视器
	watcher, err := NewConfigWatcher(configFile, func(cfg *SystemConfig) {
		callbackCalled = true
		newConfig = cfg
	})
	assert.NoError(t, err)

	// 启动监视
	watcher.Start()
	defer watcher.Stop()

	// 修改配置文件
	updatedContent := []byte(`
        node_id: "reload_test"
        meta_server: "localhost:9090"  # 修改了端口
        data_dir: "./new_data"         # 修改了数据目录
        chunk_size: 2048               # 修改了块大小
        replicas: 3                    # 修改了副本数
    `)

	// 确保写入新内容
	time.Sleep(100 * time.Millisecond)
	err = os.WriteFile(configFile, updatedContent, 0644)
	assert.NoError(t, err)

	// 手动触发检查和重新加载
	watcher.checkAndReload()

	// 验证回调是否被调用，配置是否被更新
	assert.True(t, callbackCalled, "配置变化回调应该被调用")
	if callbackCalled {
		assert.Equal(t, "reload_test", newConfig.NodeID)
		assert.Equal(t, "localhost:9090", newConfig.MetaServer)
		assert.Equal(t, "./new_data", newConfig.DataDir)
		assert.Equal(t, 2048, newConfig.ChunkSize)
		assert.Equal(t, 3, newConfig.Replicas)
	}
}

// TestLoadJSONConfig 测试加载JSON格式配置
func TestLoadJSONConfig(t *testing.T) {
	// 设置环境变量
	os.Setenv("NODE_ID", "json_node")

	// 创建JSON配置文件
	jsonFile := "config.json"
	jsonContent := []byte(`{
        "node_id": "test_node",
        "meta_server": "localhost:9000",
        "data_dir": "./json_data",
        "chunk_size": 2048,
        "replicas": 3
    }`)

	err := os.WriteFile(jsonFile, jsonContent, 0644)
	assert.NoError(t, err)
	defer os.Remove(jsonFile)

	// 使用LoadConfigJSON加载
	cfg, err := LoadConfigJSON(jsonFile)
	assert.NoError(t, err)

	// 环境变量应该覆盖配置文件
	assert.Equal(t, "json_node", cfg.NodeID)
	assert.Equal(t, "localhost:9000", cfg.MetaServer)
	assert.Equal(t, "./json_data", cfg.DataDir)
	assert.Equal(t, 2048, cfg.ChunkSize)
	assert.Equal(t, 3, cfg.Replicas)
}

// TestLoadTOMLConfig 测试加载TOML格式配置
func TestLoadTOMLConfig(t *testing.T) {
	// 设置环境变量
	os.Setenv("NODE_ID", "toml_node")

	// 创建TOML配置文件
	tomlFile := "config.toml"
	tomlContent := []byte(`
        node_id = "test_node"
        meta_server = "localhost:9500"
        data_dir = "./toml_data"
        chunk_size = 4096
        replicas = 4
        
        [logging]
        level = "debug"
        console = true
        file = "logs/toml.log"
    `)

	err := os.WriteFile(tomlFile, tomlContent, 0644)
	assert.NoError(t, err)
	defer os.Remove(tomlFile)

	// 使用LoadConfigTOML加载
	cfg, err := LoadConfigTOML(tomlFile)
	assert.NoError(t, err)

	// 环境变量应该覆盖配置文件
	assert.Equal(t, "toml_node", cfg.NodeID)
	assert.Equal(t, "localhost:9500", cfg.MetaServer)
	assert.Equal(t, "./toml_data", cfg.DataDir)
	assert.Equal(t, 4096, cfg.ChunkSize)
	assert.Equal(t, 4, cfg.Replicas)
	assert.Equal(t, "debug", cfg.Logging.Level)
}

// TestLoadConfigAuto 测试自动选择解析器
func TestLoadConfigAuto(t *testing.T) {
	// 禁用环境变量覆盖
    DisableEnvOverrideForTests()
    defer EnableEnvOverrideForTests() // 测试结束后恢复

    // 先清空环境变量，防止影响测试
    os.Unsetenv("NODE_ID")

	// 创建不同格式的配置文件
	yamlFile := "config.yaml"
	jsonFile := "config.json"
	tomlFile := "config.toml"
	unknownFile := "config.xyz"

	// 设置必需的环境变量
    os.Setenv("NODE_ID", "auto_test")

    // 创建各格式的配置文件 - 添加必要的字段
    yamlContent := []byte(`
        node_id: "yaml_node"
        meta_server: "localhost:8080"
        data_dir: "./data"
        chunk_size: 1024
        replicas: 2
    `)

    jsonContent := []byte(`{
        "node_id": "json_node",
        "meta_server": "localhost:8080",
        "data_dir": "./data",
        "chunk_size": 1024,
        "replicas": 2
    }`)

    tomlContent := []byte(`
        node_id = "toml_node"
        meta_server = "localhost:8080"
        data_dir = "./data"
        chunk_size = 1024
        replicas = 2
    `)

	err := os.WriteFile(yamlFile, yamlContent, 0644)
	assert.NoError(t, err)
	defer os.Remove(yamlFile)

	err = os.WriteFile(jsonFile, jsonContent, 0644)
	assert.NoError(t, err)
	defer os.Remove(jsonFile)

	err = os.WriteFile(tomlFile, tomlContent, 0644)
	assert.NoError(t, err)
	defer os.Remove(tomlFile)

	err = os.WriteFile(unknownFile, []byte(`random content`), 0644)
	assert.NoError(t, err)
	defer os.Remove(unknownFile)

	// 测试自动加载不同格式
	yamlCfg, err := LoadConfigAuto(yamlFile)
	assert.NoError(t, err)
	assert.Equal(t, "yaml_node", yamlCfg.NodeID)

	jsonCfg, err := LoadConfigAuto(jsonFile)
	assert.NoError(t, err)
	assert.Equal(t, "json_node", jsonCfg.NodeID)

	tomlCfg, err := LoadConfigAuto(tomlFile)
	assert.NoError(t, err)
	assert.Equal(t, "toml_node", tomlCfg.NodeID)

	// 测试不支持的格式
	_, err = LoadConfigAuto(unknownFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的配置文件格式")
}

// 设置环境变量
func setTestEnvVars() {
	os.Setenv("NODE_ID", "node_1")
	os.Setenv("META_ADDR", "192.168.1.100:8080")
	os.Setenv("DATA_DIR", "/tmp/data")
	os.Setenv("CHUNK_SIZE", "2048")
	os.Setenv("REPLICAS", "3")
}

// 创建模拟的配置文件
func createTestConfigFile(configFile string) error {
	yamlContent := []byte(`
        node_id: "temp_node"
        meta_server: "localhost:8080"
        data_dir: "./temp"
        chunk_size: 1024
        replicas: 2
    `)

	return os.WriteFile(configFile, yamlContent, 0644)
}
