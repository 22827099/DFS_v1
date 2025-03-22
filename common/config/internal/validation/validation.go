package validation

import (
    "fmt"
    "os"
    "reflect"

    "github.com/go-playground/validator/v10"
)

// ValidateConfig 验证配置是否有效
func ValidateConfig(config interface{}) error {
    validate := validator.New()
    
    // 注册自定义验证函数
    validate.RegisterValidation("path_exists", validatePathExists)
    
    // 执行基本验证
    if err := validate.Struct(config); err != nil {
        return fmt.Errorf("配置校验失败: %v", err)
    }
    
    // 使用反射检查关键字段
    v := reflect.ValueOf(config)
    if v.Kind() == reflect.Ptr {
        v = v.Elem()
    }
    
    if v.Kind() != reflect.Struct {
        return fmt.Errorf("配置必须是结构体或结构体指针")
    }
    
    // 尝试获取并验证关键字段
    nodeIDField := v.FieldByName("NodeID")
    if nodeIDField.IsValid() && nodeIDField.Type().Kind() == reflect.String && nodeIDField.String() == "" {
        return fmt.Errorf("节点ID不能为空")
    }
    
    chunkSizeField := v.FieldByName("ChunkSize")
    if chunkSizeField.IsValid() && chunkSizeField.Type().Kind() == reflect.Int && chunkSizeField.Int() < 512 {
        return fmt.Errorf("块大小不能小于512字节")
    }
    
    replicasField := v.FieldByName("Replicas")
    if replicasField.IsValid() && replicasField.Type().Kind() == reflect.Int && replicasField.Int() < 1 {
        return fmt.Errorf("副本数不能小于1")
    }
    
    return nil
}

// validatePathExists 自定义验证器，检查路径是否存在
func validatePathExists(fl validator.FieldLevel) bool {
    path := fl.Field().String()
    if path == "" {
        return true // 空路径暂时视为有效
    }
    
    _, err := os.Stat(path)
    return err == nil
}