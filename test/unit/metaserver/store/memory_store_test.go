package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/22827099/DFS_v1/internal/metaserver/core/metadata"
	"github.com/22827099/DFS_v1/internal/metaserver/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MetaServerStoreTest 测试元数据存储功能
func TestMetaServerStore(t *testing.T) {
	// 基本的CRUD操作测试
	t.Run("CreateFileTest", func(t *testing.T) {
		// 创建存储实例
		store, err := server.NewMemoryStore()
		require.NoError(t, err)
		require.NoError(t, store.Initialize())

		// 准备测试数据
		fileInfo := metadata.FileInfo{
			Path:      "/test.txt",
			Name:      "test.txt",
			Size:      1024,
			MimeType:  "text/plain",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// 创建文件
		result, err := store.CreateFile(context.Background(), fileInfo)
		require.NoError(t, err)
		assert.Equal(t, fileInfo.Path, result.Path)
		assert.Equal(t, fileInfo.Name, result.Name)
		assert.Equal(t, fileInfo.Size, result.Size)
		assert.Equal(t, fileInfo.MimeType, result.MimeType)
	})

	t.Run("ReadFileTest", func(t *testing.T) {
		// 创建存储实例并初始化
		store, err := server.NewMemoryStore()
		require.NoError(t, err)
		require.NoError(t, store.Initialize())

		// 创建测试文件
		fileInfo := metadata.FileInfo{
			Path:     "/read_test.txt",
			Name:     "read_test.txt",
			Size:     2048,
			MimeType: "text/plain",
		}
		_, err = store.CreateFile(context.Background(), fileInfo)
		require.NoError(t, err)

		// 读取文件
		result, err := store.GetFileInfo(context.Background(), "/read_test.txt")
		require.NoError(t, err)
		assert.Equal(t, fileInfo.Path, result.Path)
		assert.Equal(t, fileInfo.Size, result.Size)

		// 测试读取不存在的文件
		_, err = store.GetFileInfo(context.Background(), "/nonexistent.txt")
		assert.Error(t, err)
	})

	t.Run("UpdateFileTest", func(t *testing.T) {
		// 创建存储实例并初始化
		store, err := server.NewMemoryStore()
		require.NoError(t, err)
		require.NoError(t, store.Initialize())

		// 创建测试文件
		fileInfo := metadata.FileInfo{
			Path:     "/update_test.txt",
			Name:     "update_test.txt",
			Size:     1024,
			MimeType: "text/plain",
		}
		_, err = store.CreateFile(context.Background(), fileInfo)
		require.NoError(t, err)

		// 更新文件
		updates := map[string]interface{}{
			"size":      int64(4096),
			"mime_type": "application/octet-stream",
		}

		result, err := store.UpdateFile(context.Background(), "/update_test.txt", updates)
		require.NoError(t, err)
		assert.Equal(t, int64(4096), result.Size)
		assert.Equal(t, "application/octet-stream", result.MimeType)

		// 验证更新后的文件信息
		updated, err := store.GetFileInfo(context.Background(), "/update_test.txt")
		require.NoError(t, err)
		assert.Equal(t, int64(4096), updated.Size)
		assert.Equal(t, "application/octet-stream", updated.MimeType)
	})

	t.Run("DeleteFileTest", func(t *testing.T) {
		// 创建存储实例并初始化
		store, err := server.NewMemoryStore()
		require.NoError(t, err)
		require.NoError(t, store.Initialize())

		// 创建测试文件
		fileInfo := metadata.FileInfo{
			Path:     "/delete_test.txt",
			Name:     "delete_test.txt",
			Size:     1024,
			MimeType: "text/plain",
		}
		_, err = store.CreateFile(context.Background(), fileInfo)
		require.NoError(t, err)

		// 删除文件
		err = store.DeleteFile(context.Background(), "/delete_test.txt")
		require.NoError(t, err)

		// 验证文件已删除
		_, err = store.GetFileInfo(context.Background(), "/delete_test.txt")
		assert.Error(t, err)

		// 测试删除不存在的文件
		err = store.DeleteFile(context.Background(), "/nonexistent.txt")
		assert.Error(t, err)
	})

	t.Run("ListDirectoryTest", func(t *testing.T) {
		// 创建存储实例并初始化
		store, err := server.NewMemoryStore()
		require.NoError(t, err)
		require.NoError(t, store.Initialize())

		// 创建测试目录
		dirInfo := metadata.DirectoryInfo{
			Path: "/test_dir",
			Name: "test_dir",
		}
		_, err = store.CreateDirectory(context.Background(), dirInfo)
		require.NoError(t, err)

		// 创建测试文件
		file1 := metadata.FileInfo{
			Path:     "/test_dir/file1.txt",
			Name:     "file1.txt",
			Size:     1024,
			MimeType: "text/plain",
		}
		_, err = store.CreateFile(context.Background(), file1)
		require.NoError(t, err)

		file2 := metadata.FileInfo{
			Path:     "/test_dir/file2.txt",
			Name:     "file2.txt",
			Size:     2048,
			MimeType: "text/plain",
		}
		_, err = store.CreateFile(context.Background(), file2)
		require.NoError(t, err)

		// 创建子目录
		subDirInfo := metadata.DirectoryInfo{
			Path: "/test_dir/sub_dir",
			Name: "sub_dir",
		}
		_, err = store.CreateDirectory(context.Background(), subDirInfo)
		require.NoError(t, err)

		// 列出目录内容(非递归)
		entries, err := store.ListDirectory(context.Background(), "/test_dir", false, 0)
		require.NoError(t, err)
		assert.Equal(t, 3, len(entries))

		// 验证排序和内容
		fileCount := 0
		dirCount := 0
		for _, entry := range entries {
			if entry.IsDir {
				dirCount++
				assert.Equal(t, "sub_dir", entry.Name)
			} else {
				fileCount++
				assert.Contains(t, []string{"file1.txt", "file2.txt"}, entry.Name)
			}
		}
		assert.Equal(t, 1, dirCount)
		assert.Equal(t, 2, fileCount)

		// 测试限制数量
		limitedEntries, err := store.ListDirectory(context.Background(), "/test_dir", false, 1)
		require.NoError(t, err)
		assert.Equal(t, 1, len(limitedEntries))
	})
}
