package reflection

import (
    "fmt"
    "reflect"
    "strconv"
)

// IsZeroValue 判断字段是否为零值
func IsZeroValue(v reflect.Value) bool {
    switch v.Kind() {
    case reflect.String:
        return v.String() == ""
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        return v.Int() == 0
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        return v.Uint() == 0
    case reflect.Float32, reflect.Float64:
        return v.Float() == 0
    case reflect.Bool:
        return !v.Bool()
    case reflect.Slice, reflect.Map:
        return v.Len() == 0
    case reflect.Ptr, reflect.Interface:
        return v.IsNil()
    case reflect.Struct:
        return false // 结构体需单独处理
    default:
        return false
    }
}

// SetFieldFromString 根据字段类型从字符串设置值
func SetFieldFromString(field reflect.Value, value string) error {
    if !field.CanSet() {
        return nil
    }
    
    switch field.Kind() {
    case reflect.String:
        field.SetString(value)
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        if val, err := strconv.ParseInt(value, 10, 64); err == nil {
            field.SetInt(val)
        } else {
            return fmt.Errorf("无法转换为整数: %v", err)
        }
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        if val, err := strconv.ParseUint(value, 10, 64); err == nil {
            field.SetUint(val)
        } else {
            return fmt.Errorf("无法转换为无符号整数: %v", err)
        }
    case reflect.Float32, reflect.Float64:
        if val, err := strconv.ParseFloat(value, 64); err == nil {
            field.SetFloat(val)
        } else {
            return fmt.Errorf("无法转换为浮点数: %v", err)
        }
    case reflect.Bool:
        if val, err := strconv.ParseBool(value); err == nil {
            field.SetBool(val)
        } else {
            return fmt.Errorf("无法转换为布尔值: %v", err)
        }
    default:
        return fmt.Errorf("不支持的字段类型: %s", field.Kind())
    }
    return nil
}

// 其他反射工具函数...