package metadata

import (
	"errors"
	"sync"
	"time"
	"fmt"

	"github.com/google/uuid"  // 添加uuid包导入
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus" 
)

// ...existing code...

type ChunkInfo struct {
	// 块信息
	ChunkID string
	Offset  int64
	Size    int64
}

type FileMeta struct {
	ID        string      // 全局唯一文件ID
	Chunks    []ChunkInfo // 块分布信息
	Version   int64       // 版本号（每次修改递增）
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Storage 元数据存储接口
type Storage interface {
    // Save 保存文件元数据
    Save(meta FileMeta) error
    
    // Get 获取文件元数据
    Get(fileID string) (FileMeta, error)
    
    // Delete 删除文件元数据
    Delete(fileID string) error
    
    // List 列出符合前缀的所有文件元数据
    List(prefix string) ([]FileMeta, error)
}

type MetaManager struct {
	// 依赖注入字段
	store  Storage // 存储接口（后续实现）
	logger logging.Logger  // 日志接口

	// 并发控制
	globalLock sync.RWMutex // 全局目录锁
	fileLocks  KeyLocker    // 文件级细粒度锁（实现或使用第三方库）

	// ...其他成员变量...
	cache      *cache.Cache // 使用 go-cache 库

}

var (
	ErrFileNotFound        = errors.New("文件未找到")
	ErrVersionConflict     = errors.New("版本冲突")
	ErrInvalidChunkInfo    = errors.New("无效的块信息")
	ErrFileAlreadyExists   = errors.New("文件已存在")
	ErrInternalServerError = errors.New("内部服务器错误")
	ErrInvalidFileID       = errors.New("无效的文件ID")
	ErrInvalidMetadata     = errors.New("无效的元数据")
	ErrPermissionDenied    = errors.New("权限拒绝")
)


// CreateFile 创建一个新的文件。
// 
// 参数:
// - meta: 文件元数据，包含文件的唯一标识ID、文件块信息等。
// 
// 返回:
// - FileMeta: 成功创建的文件元数据。
// - error: 如果发生错误，返回错误信息。可能的错误包括文件块信息无效、文件已存在、保存失败等。
//
// 步骤:
// 1. 获取全局写锁，以确保线程安全。
// 2. 如果文件元数据中未提供唯一ID，则生成一个新的唯一ID。
// 3. 验证每个文件块的信息，确保块的大小大于0且偏移量不小于0。
// 4. 检查文件是否已存在，如果已存在，返回文件已存在错误。
// 5. 将文件元数据保存到存储中。
// 6. 更新内存缓存，存储文件元数据。
// 7. 记录操作日志，标记创建文件的操作。
func (m *MetaManager) CreateFile(meta FileMeta) (FileMeta, error) {
	// 1. 获取全局写锁
	m.globalLock.Lock() 
	defer m.globalLock.Unlock()

	// 2. 生成唯一ID（如果未提供）
	if meta.ID == "" {
		meta.ID = uuid.New().String()  // 使用Google的uuid包生成唯一ID
	}

	// 3. 验证块信息有效性
	for _, chunk := range meta.Chunks {
		if chunk.Size <= 0 || chunk.Offset < 0 {
			return FileMeta{}, fmt.Errorf("无效的块信息，ChunkID: %s, Size: %d, Offset: %d", chunk.ChunkID, chunk.Size, chunk.Offset)
		}
	}

	// 4. 检查文件是否已存在
	_, err := m.GetFile(meta.ID)
	if err == nil {
		return FileMeta{}, ErrFileAlreadyExists  // 如果文件已存在，返回文件已存在错误
	}

	// 5. 写入存储
	err = m.store.Save(meta)  // 调用存储接口的保存方法
	if err != nil {
		return FileMeta{}, fmt.Errorf("保存文件失败: %w", err)  // 包装原始错误
	}

	// 6. 更新内存缓存（假设有一个内存缓存）
	m.cache.Add(meta.ID, meta, 5*time.Minute)

	// 7. 记录事务日志
	m.logger.Log("创建文件", meta.ID)

	return meta, nil
}

// 获取文件元数据
func (m *MetaManager) GetFile(fileID string) (FileMeta, error) {
	m.globalLock.RLock()
	defer m.globalLock.RUnlock()

	// 查找文件元数据
	// 省略查找操作的实现
	return FileMeta{}, nil
}

// 更新文件元数据
func (m *MetaManager) UpdateFile(fileID string, updater func(FileMeta) (FileMeta, error)) error {
	m.fileLocks.Lock(fileID)
	defer m.fileLocks.Unlock(fileID)

	// 获取当前文件元数据
	fileMeta, err := m.GetFile(fileID)
	if err != nil {
		return err
	}

	// 执行更新操作
	updatedMeta, err := updater(fileMeta)
	if err != nil {
		return err
	}

	// 存储更新后的文件元数据
	// 省略存储操作的实现

	return nil
}

// 删除文件元数据
func (m *MetaManager) DeleteFile(fileID string) error {
	m.globalLock.Lock()
	defer m.globalLock.Unlock()

	// 写入操作日志
	m.logger.Log("delete file", fileID)

	// 检查文件是否存在
	_, err := m.GetFile(fileID)
	if err != nil {
		return ErrFileNotFound
	}

	// 删除文件元数据
	// 省略删除操作的实现

	return nil
}

// 列出文件元数据
func (m *MetaManager) ListFiles(prefix string) ([]FileMeta, error) {
	m.globalLock.RLock()
	defer m.globalLock.RUnlock()

	// 查找文件
	// 省略查找操作的实现
	return nil, nil
}

// ...existing code...
