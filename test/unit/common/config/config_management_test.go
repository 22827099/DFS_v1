package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"
	"reflect"

	"github.com/22827099/DFS_v1/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 设置环境变量辅助函数
func setTestEnvVars() {
    os.Setenv("NODE_ID", "node_1")
    os.Setenv("META_ADDR", "192.168.1.100:8080")
    os.Setenv("DATA_DIR", "/tmp/data")
    os.Setenv("CHUNK_SIZE", "2048")
    os.Setenv("REPLICAS", "3")
}

// 创建测试配置文件辅助函数
func createTestConfigFile(configFile string, content []byte) error {
    if content == nil {
        content = []byte(`
node_id: "temp_node"
meta_server: "localhost:8080"
data_dir: "./temp"
chunk_size: 1024
replicas: 2
`)
    }
    return os.WriteFile(configFile, content, 0644)
}

// TestConfigManagement 测试配置管理功能
func TestConfigManagement(t *testing.T) {
    // 基本配置加载测试
    t.Run("BasicConfigLoadTest", func(t *testing.T) {
        // 创建测试目录
        testDir, err := os.MkdirTemp("", "config-test-*")
        require.NoError(t, err)
        defer os.RemoveAll(testDir)
        
        // 设置环境变量
        setTestEnvVars()
        
        // 创建配置文件
        configFile := filepath.Join(testDir, "config.yaml")
        err = createTestConfigFile(configFile, nil)
        require.NoError(t, err)
        
        // 加载配置
        cfg, err := config.LoadConfig(configFile)
        require.NoError(t, err)
        
        // 验证环境变量覆盖
        assert.Equal(t, "node_1", cfg.NodeID)
        assert.Equal(t, "192.168.1.100:8080", cfg.MetaServer)
        assert.Equal(t, "/tmp/data", cfg.DataDir)
        assert.Equal(t, 2048, cfg.ChunkSize)
        assert.Equal(t, 3, cfg.Replicas)
    })

    // 测试配置文件不存在情况
    t.Run("ConfigFileNotExistsTest", func(t *testing.T) {
        // 清除环境变量，确保使用默认值
        os.Unsetenv("META_ADDR")
        os.Unsetenv("DATA_DIR")
        os.Unsetenv("CHUNK_SIZE")
        os.Unsetenv("REPLICAS")
        
        // 必须设置 NODE_ID，因为它是必需的
        os.Setenv("NODE_ID", "default_node")
        
        // 尝试加载不存在的文件
        cfg, err := config.LoadConfig("non_existent_config.yaml")
        
        // 应该不会返回错误，而是使用默认值
        assert.NoError(t, err)
        assert.NotNil(t, cfg)
        
        // 验证是否使用了默认值
        assert.Equal(t, "default_node", cfg.NodeID)
        assert.Equal(t, "localhost:8080", cfg.MetaServer)
        assert.Equal(t, "./data", cfg.DataDir)
        assert.Equal(t, 1024, cfg.ChunkSize)
        assert.Equal(t, 2, cfg.Replicas)
    })

    // 测试无效配置格式
    t.Run("InvalidConfigFormatTest", func(t *testing.T) {
        // 创建测试目录
        testDir, err := os.MkdirTemp("", "config-test-*")
        require.NoError(t, err)
        defer os.RemoveAll(testDir)
        
        // 设置必需的环境变量
        os.Setenv("NODE_ID", "test_node")
        
        // 创建格式错误的YAML文件
        invalidFile := filepath.Join(testDir, "invalid_config.yaml")
        invalidContent := []byte(`
node_id: "test_node"
meta_server: "localhost:8080
data_dir: "./data"
`) // 注意这里故意少了一个引号
        
        err = os.WriteFile(invalidFile, invalidContent, 0644)
        require.NoError(t, err)
        
        // 尝试加载无效格式的配置
        _, err = config.LoadConfig(invalidFile)
        
        // 应该返回解析错误
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "YAML解析失败")
    })

    // 测试配置验证
    t.Run("ConfigValidationTest", func(t *testing.T) {
        // 创建测试目录
        testDir, err := os.MkdirTemp("", "config-test-*")
        require.NoError(t, err)
        defer os.RemoveAll(testDir)
        
        // 子测试：测试有效配置
        t.Run("ValidConfig", func(t *testing.T) {
            validCfg := &config.SystemConfig{
                NodeID:     "test-node",
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
            err := config.ValidateConfig(validCfg)
            assert.NoError(t, err)
        })
        
        // 子测试：块大小不合法
        t.Run("InvalidChunkSize", func(t *testing.T) {
            // 创建验证会失败的配置文件
            invalidConfigFile := filepath.Join(testDir, "invalid_chunk.yaml")
            invalidContent := []byte(`
node_id: "test_node"
meta_server: "localhost:8080"
data_dir: "./data"
chunk_size: 100  # 小于最小值512
replicas: 2
`)
            
            err = os.WriteFile(invalidConfigFile, invalidContent, 0644)
            require.NoError(t, err)
            
            // 尝试加载配置
            _, err = config.LoadConfig(invalidConfigFile)
            
            // 应该返回验证错误
            assert.Error(t, err)
            assert.Contains(t, err.Error(), "块大小不能小于512字节")
        })
        
        // 子测试：副本数不合法
        t.Run("InvalidReplicaCount", func(t *testing.T) {
            invalidConfigFile := filepath.Join(testDir, "invalid_replica.yaml")
            invalidContent := []byte(`
node_id: "test_node"
meta_server: "localhost:8080"
data_dir: "./data"
chunk_size: 1024
replicas: 0      # 小于最小值1
`)
            
            err = os.WriteFile(invalidConfigFile, invalidContent, 0644)
            require.NoError(t, err)
            
            // 尝试加载配置
            _, err = config.LoadConfig(invalidConfigFile)
            
            // 应该返回验证错误
            assert.Error(t, err)
            assert.Contains(t, err.Error(), "副本数不能小于1")
        })
        
        // 子测试：节点ID为空
        t.Run("EmptyNodeID", func(t *testing.T) {
            invalidNodeIDCfg := &config.SystemConfig{
                NodeID:     "",
                MetaServer: "localhost:8080",
                DataDir:    "/var/data",
                ChunkSize:  1024,
                Replicas:   3,
            }
            err = config.ValidateConfig(invalidNodeIDCfg)
            assert.Error(t, err)
            assert.Contains(t, err.Error(), "节点ID不能为空")
        })
    })

    // 测试默认值应用
    t.Run("DefaultValuesTest", func(t *testing.T) {
        // 创建测试目录
        testDir, err := os.MkdirTemp("", "config-test-*")
        require.NoError(t, err)
        defer os.RemoveAll(testDir)
        
        // 清除环境变量
        os.Unsetenv("META_ADDR")
        os.Unsetenv("DATA_DIR")
        os.Unsetenv("CHUNK_SIZE")
        os.Unsetenv("REPLICAS")
        
        // 只提供必需的NODE_ID
        os.Setenv("NODE_ID", "minimal_node")
        
        // 创建最小配置文件，只包含必需字段
        minimalFile := filepath.Join(testDir, "minimal_config.yaml")
        minimalContent := []byte(`
node_id: "minimal_node"
`)
        
        err = os.WriteFile(minimalFile, minimalContent, 0644)
        require.NoError(t, err)
        
        // 加载配置
        cfg, err := config.LoadConfig(minimalFile)
        require.NoError(t, err)
        
        // 验证默认值是否正确应用
        assert.Equal(t, "minimal_node", cfg.NodeID)
        assert.Equal(t, "localhost:8080", cfg.MetaServer)
        assert.Equal(t, "./data", cfg.DataDir)
        assert.Equal(t, 1024, cfg.ChunkSize)
        assert.Equal(t, 2, cfg.Replicas)
        
        // 验证日志配置的默认值
        assert.Equal(t, "info", cfg.Logging.Level)
        assert.Equal(t, true, cfg.Logging.Console)
        assert.Equal(t, "logs/app.log", cfg.Logging.File)
    })

    // 测试多种格式配置文件
    t.Run("MultiFormatConfigTest", func(t *testing.T) {
        // 准备测试目录
        testDir, err := os.MkdirTemp("", "config-test-*")
        require.NoError(t, err)
        defer os.RemoveAll(testDir)

        // 准备不同格式的配置文件
        yamlConfig := filepath.Join(testDir, "config.yaml")
        jsonConfig := filepath.Join(testDir, "config.json")
        tomlConfig := filepath.Join(testDir, "config.toml")

        // 1. YAML 配置
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
        err = os.WriteFile(yamlConfig, yamlContent, 0644)
        require.NoError(t, err)

        // 2. JSON 配置
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
        err = os.WriteFile(jsonConfig, jsonContent, 0644)
        require.NoError(t, err)

        // 3. TOML 配置
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
        err = os.WriteFile(tomlConfig, tomlContent, 0644)
        require.NoError(t, err)

        // 禁用环境变量覆盖，确保测试结果一致
        config.DisableEnvOverrideForTests()
        defer config.EnableEnvOverrideForTests()

        // 测试加载YAML配置
        yamlCfg, err := config.LoadConfig(yamlConfig)
        assert.NoError(t, err)
        assert.Equal(t, "yaml-node", yamlCfg.NodeID)
        assert.Equal(t, "yaml-server:8080", yamlCfg.MetaServer)
        assert.Equal(t, "/var/data/yaml", yamlCfg.DataDir)
        assert.Equal(t, 2048, yamlCfg.ChunkSize)
        assert.Equal(t, 3, yamlCfg.Replicas)
        assert.Equal(t, "debug", yamlCfg.Logging.Level)

        // 测试加载JSON配置
        jsonCfg, err := config.LoadConfigJSON(jsonConfig)
        assert.NoError(t, err)
        assert.Equal(t, "json-node", jsonCfg.NodeID)
        assert.Equal(t, "json-server:9090", jsonCfg.MetaServer)
        assert.Equal(t, "/var/data/json", jsonCfg.DataDir)
        assert.Equal(t, 4096, jsonCfg.ChunkSize)
        assert.Equal(t, 4, jsonCfg.Replicas)
        assert.Equal(t, "info", jsonCfg.Logging.Level)

        // 测试加载TOML配置
        tomlCfg, err := config.LoadConfigTOML(tomlConfig)
        assert.NoError(t, err)
        assert.Equal(t, "toml-node", tomlCfg.NodeID)
        assert.Equal(t, "toml-server:7070", tomlCfg.MetaServer)
        assert.Equal(t, "/var/data/toml", tomlCfg.DataDir)
        assert.Equal(t, 8192, tomlCfg.ChunkSize)
        assert.Equal(t, 5, tomlCfg.Replicas)
        assert.Equal(t, "warn", tomlCfg.Logging.Level)
    })

    // 测试自动检测配置格式
    t.Run("AutoDetectFormatTest", func(t *testing.T) {
        // 准备测试目录
        testDir, err := os.MkdirTemp("", "config-test-*")
        require.NoError(t, err)
        defer os.RemoveAll(testDir)
        
        // 禁用环境变量覆盖
        config.DisableEnvOverrideForTests()
        defer config.EnableEnvOverrideForTests()
        
        // 设置必需的环境变量
        os.Setenv("NODE_ID", "auto_test")
        
        // 创建不同格式的配置文件
        yamlFile := filepath.Join(testDir, "config.yaml")
        jsonFile := filepath.Join(testDir, "config.json")
        tomlFile := filepath.Join(testDir, "config.toml")
        unknownFile := filepath.Join(testDir, "config.xyz")
        
        // 创建各格式的配置文件
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
        
        err = os.WriteFile(yamlFile, yamlContent, 0644)
        require.NoError(t, err)
        
        err = os.WriteFile(jsonFile, jsonContent, 0644)
        require.NoError(t, err)
        
        err = os.WriteFile(tomlFile, tomlContent, 0644)
        require.NoError(t, err)
        
        err = os.WriteFile(unknownFile, []byte(`random content`), 0644)
        require.NoError(t, err)
        
        // 测试自动加载不同格式
        yamlCfg, err := config.LoadConfigAuto(yamlFile)
        assert.NoError(t, err)
        assert.Equal(t, "yaml_node", yamlCfg.NodeID)
        
        jsonCfg, err := config.LoadConfigAuto(jsonFile)
        assert.NoError(t, err)
        assert.Equal(t, "json_node", jsonCfg.NodeID)
        
        tomlCfg, err := config.LoadConfigAuto(tomlFile)
        assert.NoError(t, err)
        assert.Equal(t, "toml_node", tomlCfg.NodeID)
        
        // 测试不支持的格式
        _, err = config.LoadConfigAuto(unknownFile)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "不支持的配置文件格式")
    })

    // 测试配置热重载
    t.Run("HotReloadTest", func(t *testing.T) {
        // 创建临时配置文件
        tempFile, err := os.CreateTemp("", "config-*.yaml")
        require.NoError(t, err)
        defer os.Remove(tempFile.Name())

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
        require.NoError(t, err)
        tempFile.Close()

        // 禁用环境变量覆盖
        config.DisableEnvOverrideForTests()
        defer config.EnableEnvOverrideForTests()

        // 创建配置观察器
        configUpdated := false
        var updatedConfig *config.SystemConfig

        watcher, err := config.NewConfigWatcher(tempFile.Name(), func(cfg *config.SystemConfig) {
            configUpdated = true
            updatedConfig = cfg
        })
        require.NoError(t, err)

        // 启动监视
        watcher.Start()
        defer watcher.Stop()

        // 修改配置文件
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

        // 写入更新的配置
        err = os.WriteFile(tempFile.Name(), updatedConfigData, 0644)
        require.NoError(t, err)

		// 手动触发检查（使用反射，不依赖接口）
		method := reflect.ValueOf(watcher).MethodByName("ForceReload")
		if method.IsValid() {
			results := method.Call([]reflect.Value{})
			if len(results) > 0 && !results[0].IsNil() {
				err, ok := results[0].Interface().(error)
				if ok {
					require.NoError(t, err)
				}
			}
		} else {
			t.Fatalf("ForceReload 方法不存在")
		}

        // 验证配置是否更新
        assert.True(t, configUpdated, "配置更新回调应该被调用")
        if configUpdated {
            assert.Equal(t, "hot-reload-node", updatedConfig.NodeID)
            assert.Equal(t, "localhost:9090", updatedConfig.MetaServer)
            assert.Equal(t, "/var/new-data", updatedConfig.DataDir)
            assert.Equal(t, 2048, updatedConfig.ChunkSize)
            assert.Equal(t, 3, updatedConfig.Replicas)
            assert.Equal(t, "debug", updatedConfig.Logging.Level)
        }
    })
}