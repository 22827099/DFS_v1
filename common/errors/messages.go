package errors

// 错误码->消息模板映射
var messages = map[int]map[string]string{
	ErrFileNotFound: {
		"en": "File not found: %s",
		"zh": "文件不存在: %s",
	},
	ErrPermission: {
		"en": "Permission denied: %s",
		"zh": "权限拒绝: %s",
	},
	ErrRPCFailure: {
		"en": "RPC failure: %s",
		"zh": "RPC失败: %s",
	},
	
}
