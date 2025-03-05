package config

import (
    "os"
    "testing"

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
    assert.Equal(t, "node_1", cfg.NodeID)            // 环境变量覆盖生效
    assert.Equal(t, "192.168.1.100:8080", cfg.MetaServer)
    assert.Equal(t, "/tmp/data", cfg.DataDir)
    assert.Equal(t, 2048, cfg.ChunkSize)             // 环境变量覆盖YAML中的1024
    assert.Equal(t, 3, cfg.Replicas)                 // 环境变量覆盖YAML中的2
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