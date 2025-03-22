package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/22827099/DFS_v1/common/config"
	"github.com/22827099/DFS_v1/common/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试辅助函数 - 设置环境变量
func setupEnvVars(t *testing.T) {
	t.Helper()

	// 保存原始环境以便恢复
	t.Cleanup(func() {
		os.Unsetenv("NODE_ID")
		os.Unsetenv("META_ADDR")
		os.Unsetenv("DATA_DIR")
		os.Unsetenv("CHUNK_SIZE")
		os.Unsetenv("REPLICAS")
	})

	os.Setenv("NODE_ID", "node_1")
	os.Setenv("META_ADDR", "192.168.1.100:8080")
	os.Setenv("DATA_DIR", "/tmp/data")
	os.Setenv("CHUNK_SIZE", "2048")
	os.Setenv("REPLICAS", "3")
}

// 测试辅助函数 - 创建临时目录
func createTempDir(t *testing.T) string {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "config-test-*")
	require.NoError(t, err, "创建临时目录失败")

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return tempDir
}

// 测试辅助函数 - 创建配置文件
func createConfigFile(t *testing.T, path string, content []byte) {
	t.Helper()

	if content == nil {
		content = []byte(`
node_id: "temp_node"
meta_server: "localhost:8080"
data_dir: "./temp"
chunk_size: 1024
replicas: 2
`)
	}

	err := os.WriteFile(path, content, 0644)
	require.NoError(t, err, "创建配置文件失败")
}

// TestBasicConfigLoading 测试基本配置加载功能
func TestBasicConfigLoading(t *testing.T) {
	// 准备测试环境
	tempDir := createTempDir(t)
	setupEnvVars(t)

	// 创建配置文件
	configFile := filepath.Join(tempDir, "config.yaml")
	createConfigFile(t, configFile, nil)

	// 加载配置
	cfg, err := config.LoadSystemConfig(configFile)
	require.NoError(t, err, "加载配置应该成功")

	// 验证环境变量覆盖
	assert.Equal(t, "node_1", cfg.NodeID.String(), "NodeID 应该被环境变量覆盖")
	assert.Equal(t, "192.168.1.100:8080", cfg.MetaServer, "MetaServer 应该被环境变量覆盖")
	assert.Equal(t, "/tmp/data", cfg.DataDir, "DataDir 应该被环境变量覆盖")
	assert.Equal(t, 2048, cfg.ChunkSize, "ChunkSize 应该被环境变量覆盖")
	assert.Equal(t, 3, cfg.Replicas, "Replicas 应该被环境变量覆盖")
}

// TestDefaultValuesAndFileNotExists 测试默认值应用和文件不存在的情况
func TestDefaultValuesAndFileNotExists(t *testing.T) {
	// 清除并设置环境变量
	os.Unsetenv("META_ADDR")
	os.Unsetenv("DATA_DIR")
	os.Unsetenv("CHUNK_SIZE")
	os.Unsetenv("REPLICAS")
	os.Setenv("NODE_ID", "default_node") // 必须设置，因为它是必需的

	t.Cleanup(func() {
		os.Unsetenv("NODE_ID")
	})

	// 尝试加载不存在的文件
	cfg, err := config.LoadSystemConfig("non_existent_config.yaml")

	// 验证
	assert.NoError(t, err, "加载不存在的配置文件应该成功并使用默认值")
	assert.NotNil(t, cfg, "应该返回有效的配置对象")
	assert.Equal(t, "default_node", cfg.NodeID.String(), "NodeID 应该来自环境变量")
	assert.Equal(t, "localhost:8080", cfg.MetaServer, "MetaServer 应该使用默认值")
	assert.Equal(t, "./data", cfg.DataDir, "DataDir 应该使用默认值")
	assert.Equal(t, 1024, cfg.ChunkSize, "ChunkSize 应该使用默认值")
	assert.Equal(t, 2, cfg.Replicas, "Replicas 应该使用默认值")
	assert.Equal(t, "info", cfg.Logging.Level, "Logging.Level 应该使用默认值")
	assert.Equal(t, true, cfg.Logging.Console, "Logging.Console 应该使用默认值")
	assert.Equal(t, "logs/app.log", cfg.Logging.File, "Logging.File 应该使用默认值")
}

// TestInvalidConfig 测试无效配置格式
func TestInvalidConfig(t *testing.T) {
	tempDir := createTempDir(t)
	os.Setenv("NODE_ID", "test_node")

	t.Cleanup(func() {
		os.Unsetenv("NODE_ID")
	})

	// 创建格式错误的YAML文件
	invalidFile := filepath.Join(tempDir, "invalid_config.yaml")
	invalidContent := []byte(`
node_id: "test_node"
meta_server: "localhost:8080
data_dir: "./data"
`) // 注意这里缺少引号

	createConfigFile(t, invalidFile, invalidContent)

	// 尝试加载无效格式的配置
	_, err := config.LoadSystemConfig(invalidFile)

	// 验证
	assert.Error(t, err, "加载无效格式的配置应该返回错误")
	assert.Contains(t, err.Error(), "解析YAML配置失败", "错误消息应该指明解析失败")
}

// TestConfigValidation 测试配置验证功能
func TestConfigValidation(t *testing.T) {
	tempDir := createTempDir(t)

	// 测试有效配置
	t.Run("ValidConfig", func(t *testing.T) {
		validCfg := &config.SystemConfig{
			NodeID:     types.NodeID("test-node"),
			MetaServer: "localhost:8080",
			DataDir:    "/var/data",
			ChunkSize:  1024,
			Replicas:   3,
			Logging: config.LoggingConfig{
				Level:   "info",
				Console: true,
				File:    "logs/app.log",
			},
			Server: config.ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
		}

		validator := config.NewValidator()
		validator.AddRule("ChunkSize", true, func(v interface{}) error {
			return nil // 简化的验证
		})
		err := validator.Validate(validCfg)
		assert.NoError(t, err, "有效配置应该通过验证")
	})

	// 测试块大小不合法
	t.Run("InvalidChunkSize", func(t *testing.T) {
		invalidConfigFile := filepath.Join(tempDir, "invalid_chunk.yaml")
		invalidContent := []byte(`
node_id: "test_node"
meta_server: "localhost:8080"
data_dir: "./data"
chunk_size: 100  # 小于最小值512
replicas: 2
`)

		createConfigFile(t, invalidConfigFile, invalidContent)

		// 尝试加载配置
		_, err := config.LoadSystemConfig(invalidConfigFile)
		assert.Error(t, err, "配置验证应该失败")
	})
}

// TestMultiFormatConfig 测试多种格式配置文件
func TestMultiFormatConfig(t *testing.T) {
	tempDir := createTempDir(t)

	// 禁用环境变量覆盖，确保测试结果一致
	config.DisableEnvOverrideForTests()
	t.Cleanup(config.EnableEnvOverrideForTests)

	// 准备不同格式的配置文件
	yamlFile := filepath.Join(tempDir, "config.yaml")
	jsonFile := filepath.Join(tempDir, "config.json")
	tomlFile := filepath.Join(tempDir, "config.toml")

	yamlContent := []byte(`
node_id: "yaml-node"
meta_server: "yaml-server:8080"
data_dir: "/var/data/yaml"
chunk_size: 2048
replicas: 3
logging:
  level: "debug"
  console: true
  file: "logs/yaml.log"
`)

	jsonContent := []byte(`{
"node_id": "json-node",
"meta_server": "json-server:9090",
"data_dir": "/var/data/json",
"chunk_size": 4096,
"replicas": 4,
"logging": {
  "level": "info",
  "console": false,
  "file": "logs/json.log"
}
}`)

	tomlContent := []byte(`
node_id = "toml-node"
meta_server = "toml-server:7070"
data_dir = "/var/data/toml"
chunk_size = 8192
replicas = 5

[logging]
level = "warn"
console = true
file = "logs/toml.log"
`)

	createConfigFile(t, yamlFile, yamlContent)
	createConfigFile(t, jsonFile, jsonContent)
	createConfigFile(t, tomlFile, tomlContent)

	// 测试加载YAML配置
	t.Run("YAML", func(t *testing.T) {
		cfg, err := config.LoadSystemConfig(yamlFile)
		require.NoError(t, err, "加载YAML配置应该成功")
		assert.Equal(t, types.NodeID("yaml-node"), cfg.NodeID)
		assert.Equal(t, "yaml-server:8080", cfg.MetaServer)
		assert.Equal(t, "/var/data/yaml", cfg.DataDir)
		assert.Equal(t, 2048, cfg.ChunkSize)
		assert.Equal(t, 3, cfg.Replicas)
		assert.Equal(t, "debug", cfg.Logging.Level)
	})

	// 测试加载JSON配置
	t.Run("JSON", func(t *testing.T) {
		cfg, err := config.LoadConfigJSON(jsonFile)
		require.NoError(t, err, "加载JSON配置应该成功")
		assert.Equal(t, types.NodeID("json-node"), cfg.NodeID)
		assert.Equal(t, "json-server:9090", cfg.MetaServer)
		assert.Equal(t, "/var/data/json", cfg.DataDir)
		assert.Equal(t, 4096, cfg.ChunkSize)
		assert.Equal(t, 4, cfg.Replicas)
		assert.Equal(t, "info", cfg.Logging.Level)
	})

	// 测试加载TOML配置
	t.Run("TOML", func(t *testing.T) {
		cfg, err := config.LoadConfigTOML(tomlFile)
		require.NoError(t, err, "加载TOML配置应该成功")
		assert.Equal(t, types.NodeID("toml-node"), cfg.NodeID)
		assert.Equal(t, "toml-server:7070", cfg.MetaServer)
		assert.Equal(t, "/var/data/toml", cfg.DataDir)
		assert.Equal(t, 8192, cfg.ChunkSize)
		assert.Equal(t, 5, cfg.Replicas)
		assert.Equal(t, "warn", cfg.Logging.Level)
	})
}

// TestConfigAutoDetection 测试配置自动检测功能
func TestConfigAutoDetection(t *testing.T) {
	tempDir := createTempDir(t)

	// 禁用环境变量覆盖
	config.DisableEnvOverrideForTests()
	t.Cleanup(config.EnableEnvOverrideForTests)

    // 创建不同格式的配置文件
        yamlFile := filepath.Join(tempDir, "config.yaml")
        jsonFile := filepath.Join(tempDir, "config.json")
        tomlFile := filepath.Join(tempDir, "config.toml")
        unknownFile := filepath.Join(tempDir, "config.xyz")
    
        createConfigFile(t, yamlFile, []byte(`node_id: "yaml_node"`))
        createConfigFile(t, jsonFile, []byte(`{"node_id": "json_node"}`))
        // 修复TOML格式，增加更完整的结构以确保正确解析
        createConfigFile(t, tomlFile, []byte(`
node_id = "toml_node"
meta_server = "localhost:8080"
data_dir = "./data"
chunk_size = 1024
replicas = 2
`))
        createConfigFile(t, unknownFile, []byte(`random content`))
    
        // 测试各种格式的自动检测
        t.Run("AutoYAML", func(t *testing.T) {
            cfg, err := config.LoadConfigAuto(yamlFile)
            require.NoError(t, err, "自动检测YAML应该成功")
            assert.Equal(t, types.NodeID("yaml_node"), cfg.NodeID)
        })
    
        t.Run("AutoJSON", func(t *testing.T) {
            cfg, err := config.LoadConfigAuto(jsonFile)
            require.NoError(t, err, "自动检测JSON应该成功")
            assert.Equal(t, types.NodeID("json_node"), cfg.NodeID)
        })
    
        t.Run("AutoTOML", func(t *testing.T) {
            cfg, err := config.LoadConfigAuto(tomlFile)
            require.NoError(t, err, "自动检测TOML应该成功")
            assert.Equal(t, types.NodeID("toml_node"), cfg.NodeID)
        })

	t.Run("UnknownFormat", func(t *testing.T) {
		_, err := config.LoadConfigAuto(unknownFile)
		assert.Error(t, err, "不支持的格式应该返回错误")
		assert.Contains(t, err.Error(), "不支持的配置文件格式")
	})
}

// TestConfigHotReload 测试配置热重载功能
func TestConfigHotReload(t *testing.T) {
	// 创建临时配置文件
	tempFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err, "创建临时文件失败")
	tempFileName := tempFile.Name()

	t.Cleanup(func() {
		os.Remove(tempFileName)
	})

	// 写入初始配置
	initialConfig := []byte(`
node_id: "hot-reload-node"
meta_server: "localhost:8080"
data_dir: "/var/data"
chunk_size: 1024
replicas: 2
logging:
  level: "info"
  console: true
  file: "logs/app.log"
`)

	_, err = tempFile.Write(initialConfig)
	require.NoError(t, err, "写入初始配置失败")
	tempFile.Close()

	// 禁用环境变量覆盖
	config.DisableEnvOverrideForTests()
	t.Cleanup(config.EnableEnvOverrideForTests)

	// 创建配置观察器
	var configUpdated bool
	var updatedConfig *config.SystemConfig

	watcher, err := config.NewConfigWatcher(tempFileName, func(cfg *config.SystemConfig) {
		configUpdated = true
		updatedConfig = cfg
	})
	require.NoError(t, err, "创建配置观察器失败")

	// 启动监视
	watcher.Start()
	t.Cleanup(watcher.Stop)

	// 更新配置文件
	updatedConfigData := []byte(`
node_id: "hot-reload-node"
meta_server: "localhost:9090"  # 修改了端口
data_dir: "/var/new-data"     # 修改了数据目录
chunk_size: 2048              # 修改了块大小
replicas: 3                   # 修改了副本数
logging:
  level: "debug"              # 修改了日志级别
  console: true
  file: "logs/app.log"
`)

	// 等待确保文件修改时间戳不同
	time.Sleep(100 * time.Millisecond)

	// 写入更新后的配置
	err = os.WriteFile(tempFileName, updatedConfigData, 0644)
	require.NoError(t, err, "更新配置文件失败")

	// 手动触发重新加载
	err = watcher.ForceReload()
	require.NoError(t, err, "强制重载配置失败")

	// 验证配置是否更新
	assert.True(t, configUpdated, "配置更新回调应该被调用")
	if configUpdated {
		assert.Equal(t, types.NodeID("hot-reload-node"), updatedConfig.NodeID)
		assert.Equal(t, "localhost:9090", updatedConfig.MetaServer)
		assert.Equal(t, "/var/new-data", updatedConfig.DataDir)
		assert.Equal(t, 2048, updatedConfig.ChunkSize)
		assert.Equal(t, 3, updatedConfig.Replicas)
		assert.Equal(t, "debug", updatedConfig.Logging.Level)
	}
}
