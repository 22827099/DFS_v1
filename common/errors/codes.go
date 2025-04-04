package errors

// ErrorCode 表示系统错误码
type ErrorCode int

const (
	// 系统级错误码 (1-999)
	Unknown           ErrorCode = 1  // 未知错误
	Internal          ErrorCode = 2  // 内部系统错误
	InvalidArgument   ErrorCode = 3  // 无效参数
	NotFound          ErrorCode = 4  // 资源不存在
	AlreadyExists     ErrorCode = 5  // 资源已存在
	PermissionDenied  ErrorCode = 6  // 权限不足
	Unauthenticated   ErrorCode = 7  // 未认证
	ResourceExhausted ErrorCode = 8  // 资源耗尽
	Unavailable       ErrorCode = 9  // 服务不可用
	Timeout           ErrorCode = 10 // 操作超时
	RateLimitExceeded ErrorCode = 11 // 速率限制超出

	// 配置错误 (1000-1099)
	ConfigParseError      ErrorCode = 1000 // 配置解析错误
	ConfigValidationError ErrorCode = 1001 // 配置验证错误

	// 网络错误 (1100-1199)
	NetworkError    ErrorCode = 1100 // 网络错误
	ConnectionError ErrorCode = 1101 // 连接错误

	// 存储错误 (1200-1299)
	StorageError      ErrorCode = 1200 // 存储通用错误
	DataCorruption    ErrorCode = 1201 // 数据损坏
	FileNotFound      ErrorCode = 1210 // 文件不存在
	FileAlreadyExists ErrorCode = 1211 // 文件已存在
	QuotaExceeded     ErrorCode = 1212 // 配额超出

	// 一致性错误 (1300-1399)
	ConsensusError    ErrorCode = 1300 // 共识错误
	QuorumNotAchieved ErrorCode = 1301 // 未达到法定人数

	// 安全错误 (1400-1499)
	SecurityError       ErrorCode = 1400 // 安全通用错误
	CryptoError         ErrorCode = 1401 // 加密/解密错误
	AuthenticationError ErrorCode = 1402 // 认证错误
	TokenError          ErrorCode = 1403 // 令牌错误

	// 分布式系统错误 (1500-1599)
	RPCFailure        ErrorCode = 1500 // RPC调用失败
	NodeUnavailable   ErrorCode = 1501 // 节点不可用
	ConsistencyFailed ErrorCode = 1502 // 一致性检查失败
	ReplicationFailed ErrorCode = 1503 // 副本复制失败
	PartitionError    ErrorCode = 1504 // 分区错误
)

// 错误码对应的文本描述映射
var codeText = map[ErrorCode]string{
	Unknown:           "未知错误",
	Internal:          "内部系统错误",
	InvalidArgument:   "无效参数",
	NotFound:          "资源不存在",
	AlreadyExists:     "资源已存在",
	PermissionDenied:  "权限不足",
	Unauthenticated:   "未认证",
	ResourceExhausted: "资源耗尽",
	Unavailable:       "服务不可用",
	Timeout:           "操作超时",
	RateLimitExceeded: "速率限制超出",

	ConfigParseError:      "配置解析错误",
	ConfigValidationError: "配置验证错误",

	NetworkError:    "网络错误",
	ConnectionError: "连接错误",

	StorageError:      "存储错误",
	DataCorruption:    "数据损坏",
	FileNotFound:      "文件不存在",
	FileAlreadyExists: "文件已存在",
	QuotaExceeded:     "配额超出",

	ConsensusError:    "共识错误",
	QuorumNotAchieved: "未达到法定人数",

	SecurityError:       "安全错误",
	CryptoError:         "加密/解密错误",
	AuthenticationError: "认证错误",
	TokenError:          "令牌错误",

	RPCFailure:        "RPC调用失败",
	NodeUnavailable:   "节点不可用",
	ConsistencyFailed: "一致性检查失败",
	ReplicationFailed: "副本复制失败",
	PartitionError:    "分区错误",
}

// Text 返回错误码对应的文本描述
func (e ErrorCode) Text() string {
	if text, ok := codeText[e]; ok {
		return text
	}
	return codeText[Unknown]
}
