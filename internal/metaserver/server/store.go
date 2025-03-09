package server

import (
	"context"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/22827099/DFS_v1/common/errors"
	"github.com/22827099/DFS_v1/internal/metaserver/core/metadata"
)

// MemoryStore 是一个基于内存的元数据存储实现
type MemoryStore struct {
	mu          sync.RWMutex
	files       map[string]*metadata.FileInfo
	directories map[string]*metadata.DirectoryInfo
	initialized bool
}

// NewMemoryStore 创建一个新的内存元数据存储
func NewMemoryStore() (*MemoryStore, error) {
	return &MemoryStore{
		files:       make(map[string]*metadata.FileInfo),
		directories: make(map[string]*metadata.DirectoryInfo),
		initialized: false,
	}, nil
}

// Initialize 初始化存储
func (s *MemoryStore) Initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return errors.New(errors.AlreadyExists, "存储已经初始化")
	}

	// 创建根目录
	rootDir := &metadata.DirectoryInfo{
		Path:      "/",
		Name:      "/",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.directories["/"] = rootDir

	s.initialized = true
	return nil
}

// Close 关闭存储
func (s *MemoryStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return nil
	}

	// 清空所有数据
	s.files = make(map[string]*metadata.FileInfo)
	s.directories = make(map[string]*metadata.DirectoryInfo)
	s.initialized = false

	return nil
}

// GetFileInfo 获取文件信息
func (s *MemoryStore) GetFileInfo(ctx context.Context, filePath string) (*metadata.FileInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, errors.New(errors.Internal, "存储未初始化")
	}

	// 规范化路径
	filePath = path.Clean(filePath)

	file, exists := s.files[filePath]
	if !exists {
		return nil, errors.New(errors.NotFound, "文件不存在")
	}

	// 返回文件信息的副本
	return cloneFileInfo(file), nil
}

// CreateFile 创建新文件
func (s *MemoryStore) CreateFile(ctx context.Context, fileInfo metadata.FileInfo) (*metadata.FileInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return nil, errors.New(errors.Internal, "存储未初始化")
	}

	// 规范化路径
	filePath := path.Clean(fileInfo.Path)
	fileInfo.Path = filePath

	// 检查文件是否已存在
	if _, exists := s.files[filePath]; exists {
		return nil, errors.New(errors.AlreadyExists, "文件已存在")
	}

	// 检查父目录是否存在
	parentDir := path.Dir(filePath)
	if parentDir != "/" {
		if _, exists := s.directories[parentDir]; !exists {
			return nil, errors.New(errors.NotFound, "父目录不存在")
		}
	}

	// 设置创建和更新时间
	now := time.Now()
	fileInfo.CreatedAt = now
	fileInfo.UpdatedAt = now

	// 如果没有设置名称，使用路径中的名称
	if fileInfo.Name == "" {
		fileInfo.Name = path.Base(filePath)
	}

	// 存储文件信息的副本
	s.files[filePath] = cloneFileInfo(&fileInfo)

	// 返回文件信息的副本
	return cloneFileInfo(s.files[filePath]), nil
}

// UpdateFile 更新文件信息
func (s *MemoryStore) UpdateFile(ctx context.Context, filePath string, updates map[string]interface{}) (*metadata.FileInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return nil, errors.New(errors.Internal, "存储未初始化")
	}

	// 规范化路径
	filePath = path.Clean(filePath)

	// 检查文件是否存在
	file, exists := s.files[filePath]
	if !exists {
		return nil, errors.New(errors.NotFound, "文件不存在")
	}

	// 更新文件信息
	for key, value := range updates {
		switch key {
		case "size":
			if size, ok := value.(int64); ok {
				file.Size = size
			}
		case "chunks":
			if chunks, ok := value.([]metadata.ChunkInfo); ok {
				file.Chunks = chunks
			}
		case "mime_type":
			if mimeType, ok := value.(string); ok {
				file.MimeType = mimeType
			}
		case "metadata":
			if meta, ok := value.(map[string]string); ok {
				file.Metadata = meta
			}
		}
	}

	// 更新修改时间
	file.UpdatedAt = time.Now()

	// 返回文件信息的副本
	return cloneFileInfo(file), nil
}

// DeleteFile 删除文件
func (s *MemoryStore) DeleteFile(ctx context.Context, filePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return errors.New(errors.Internal, "存储未初始化")
	}

	// 规范化路径
	filePath = path.Clean(filePath)

	// 检查文件是否存在
	if _, exists := s.files[filePath]; !exists {
		return errors.New(errors.NotFound, "文件不存在")
	}

	// 删除文件
	delete(s.files, filePath)

	return nil
}

// ListDirectory 列出目录内容
func (s *MemoryStore) ListDirectory(ctx context.Context, dirPath string, recursive bool, limit int) ([]metadata.DirectoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, errors.New(errors.Internal, "存储未初始化")
	}

	// 规范化路径
	dirPath = path.Clean(dirPath)
	if dirPath != "/" && dirPath[len(dirPath)-1] != '/' {
		dirPath += "/"
	}

	// 检查目录是否存在
	if _, exists := s.directories[dirPath]; !exists && dirPath != "/" {
		return nil, errors.New(errors.NotFound, "目录不存在")
	}

	var entries []metadata.DirectoryEntry
	count := 0

	// 添加子目录
	for path, dir := range s.directories {
		if count >= limit && limit > 0 {
			break
		}

		// 如果是同一路径，跳过
		if path == dirPath {
			continue
		}

		if path == "/" && dirPath != "/" {
			continue
		}

		// 检查是否是目标目录的子目录
		if dirPath == "/" || strings.HasPrefix(path, dirPath) {
			// 非递归模式下，只列出直接子目录
			if !recursive && path != dirPath && strings.Count(path[len(dirPath):], "/") > 1 {
				continue
			}

			entry := metadata.DirectoryEntry{
				Name:       dir.Name,
				Path:       dir.Path,
				IsDir:      true,
				Size:       0,
				CreatedAt:  dir.CreatedAt,
				UpdatedAt:  dir.UpdatedAt,
				ChildCount: countChildren(s, dir.Path),
			}
			entries = append(entries, entry)
			count++
		}
	}

	// 添加文件
	for filePath, file := range s.files {
		if count >= limit && limit > 0 {
			break
		}

		parentDir := path.Dir(filePath)
		if parentDir != "/" {
			parentDir += "/"
		}

		if parentDir == dirPath || (recursive && strings.HasPrefix(parentDir, dirPath)) {
			entry := metadata.DirectoryEntry{
				Name:      file.Name,
				Path:      file.Path,
				IsDir:     false,
				Size:      file.Size,
				CreatedAt: file.CreatedAt,
				UpdatedAt: file.UpdatedAt,
				MimeType:  file.MimeType,
			}
			entries = append(entries, entry)
			count++
		}
	}

	return entries, nil
}

// CreateDirectory 创建目录
func (s *MemoryStore) CreateDirectory(ctx context.Context, dirInfo metadata.DirectoryInfo) (*metadata.DirectoryInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return nil, errors.New(errors.Internal, "存储未初始化")
	}

	// 规范化路径
	dirPath := path.Clean(dirInfo.Path)
	if dirPath != "/" {
		dirPath += "/"
	}
	dirInfo.Path = dirPath

	// 检查目录是否已存在
	if _, exists := s.directories[dirPath]; exists {
		return nil, errors.New(errors.AlreadyExists, "目录已存在")
	}

	// 检查父目录是否存在
	parentDir := path.Dir(dirPath)
	if parentDir != "/" {
		parentDir += "/"
	}
	if _, exists := s.directories[parentDir]; !exists && parentDir != "/" {
		return nil, errors.New(errors.NotFound, "父目录不存在")
	}

	// 设置创建和更新时间
	now := time.Now()
	dirInfo.CreatedAt = now
	dirInfo.UpdatedAt = now

	// 如果没有设置名称，使用路径中的名称
	if dirInfo.Name == "" {
		dirInfo.Name = path.Base(dirPath)
	}

	// 存储目录信息的副本
	s.directories[dirPath] = cloneDirectoryInfo(&dirInfo)

	// 返回目录信息的副本
	return cloneDirectoryInfo(s.directories[dirPath]), nil
}

// DeleteDirectory 删除目录
func (s *MemoryStore) DeleteDirectory(ctx context.Context, dirPath string, recursive bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return errors.New(errors.Internal, "存储未初始化")
	}

	// 规范化路径
	dirPath = path.Clean(dirPath)
	if dirPath != "/" {
		dirPath += "/"
	}

	// 防止删除根目录
	if dirPath == "/" {
		return errors.New(errors.PermissionDenied, "不允许删除根目录")
	}

	// 检查目录是否存在
	if _, exists := s.directories[dirPath]; !exists {
		return errors.New(errors.NotFound, "目录不存在")
	}

	// 检查目录是否为空或者是否允许递归删除
	hasChildren := false

	// 检查是否有子目录
	for path := range s.directories {
		if path != dirPath && strings.HasPrefix(path, dirPath) {
			hasChildren = true
			if !recursive {
				return errors.New(errors.PermissionDenied, "目录不为空，需要递归删除")
			}
			break
		}
	}

	// 检查是否有文件
	if !hasChildren {
		for filePath := range s.files {
			parentDir := path.Dir(filePath)
			if parentDir != "/" {
				parentDir += "/"
			}
			if strings.HasPrefix(parentDir, dirPath) {
				hasChildren = true
				if !recursive {
					return errors.New(errors.PermissionDenied, "目录不为空，需要递归删除")
				}
				break
			}
		}
	}

	// 递归删除所有子目录和文件
	if recursive {
		// 删除子目录
		for path := range s.directories {
			if path != dirPath && strings.HasPrefix(path, dirPath) {
				delete(s.directories, path)
			}
		}

		// 删除文件
		for filePath := range s.files {
			parentDir := path.Dir(filePath)
			if parentDir != "/" {
				parentDir += "/"
			}
			if strings.HasPrefix(parentDir, dirPath) {
				delete(s.files, filePath)
			}
		}
	}

	// 删除目录本身
	delete(s.directories, dirPath)

	return nil
}

// 辅助函数

// countChildren 计算目录中的子项数量
func countChildren(s *MemoryStore, dirPath string) int {
	count := 0

	// 规范化路径
	if dirPath != "/" && dirPath[len(dirPath)-1] != '/' {
		dirPath += "/"
	}

	// 计算直接子目录
	for path := range s.directories {
		if path == dirPath {
			continue
		}
		if strings.HasPrefix(path, dirPath) {
			// 只计算直接子目录
			remaining := path[len(dirPath):]
			if !strings.Contains(remaining, "/") || remaining == "/" {
				count++
			}
		}
	}

	// 计算直接子文件
	for filePath := range s.files {
		parentDir := path.Dir(filePath)
		if parentDir != "/" {
			parentDir += "/"
		}
		if parentDir == dirPath {
			count++
		}
	}

	return count
}

// cloneFileInfo 创建FileInfo的深拷贝
func cloneFileInfo(info *metadata.FileInfo) *metadata.FileInfo {
	if info == nil {
		return nil
	}

	clone := &metadata.FileInfo{
		Path:      info.Path,
		Name:      info.Name,
		Size:      info.Size,
		MimeType:  info.MimeType,
		CreatedAt: info.CreatedAt,
		UpdatedAt: info.UpdatedAt,
	}

	if info.Metadata != nil {
		clone.Metadata = make(map[string]string)
		for k, v := range info.Metadata {
			clone.Metadata[k] = v
		}
	}

	if len(info.Chunks) > 0 {
		clone.Chunks = make([]metadata.ChunkInfo, len(info.Chunks))
		copy(clone.Chunks, info.Chunks)
	}

	return clone
}

// cloneDirectoryInfo 创建DirectoryInfo的深拷贝
func cloneDirectoryInfo(info *metadata.DirectoryInfo) *metadata.DirectoryInfo {
	if info == nil {
		return nil
	}

	clone := &metadata.DirectoryInfo{
		Path:      info.Path,
		Name:      info.Name,
		CreatedAt: info.CreatedAt,
		UpdatedAt: info.UpdatedAt,
	}

	if info.Metadata != nil {
		clone.Metadata = make(map[string]string)
		for k, v := range info.Metadata {
			clone.Metadata[k] = v
		}
	}

	return clone
}
