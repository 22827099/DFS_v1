package errors

// 错误码分类
const (
	// 客户端错误 (1000-1999)
	ErrFileNotFound      = 1001 // 文件不存在
	ErrPermission        = 1002 // 权限错误
	ErrInvalidArgument   = 1003 // 参数无效
	ErrFileAlreadyExists = 1004 // 文件已存在
	ErrQuotaExceeded     = 1005 // 配额超出

	// 服务端错误 (2000-2999)
	ErrRPCFailure        = 2001 // RPC调用失败
	ErrInternalStorage   = 2002 // 存储错误
	ErrDatabaseError     = 2003 // 数据库错误
	ErrResourceExhausted = 2004 // 资源耗尽

	// 分布式系统特有错误 (3000-3999)
	ErrNodeUnavailable   = 3001 // 节点不可用
	ErrConsistencyFailed = 3002 // 一致性检查失败
	ErrReplicationFailed = 3003 // 副本复制失败
	ErrPartitionError    = 3004 // 分区错误
)
