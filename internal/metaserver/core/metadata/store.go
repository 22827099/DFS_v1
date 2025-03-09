package metadata

import (
	"github.com/22827099/DFS_v1/common/errors"
)

// NewDefaultStore 创建默认的元数据存储实现
func NewDefaultStore() (Store, error) {
	// 实际实现会在server包中的MemoryStore
	// 此处返回错误，应由具体实现来提供实例
	return nil, errors.New(errors.Internal, "需要具体实现提供存储实例")
}
