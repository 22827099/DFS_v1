package store_test

import (
	"context"
	"testing"

	"github.com/22827099/DFS_v1/internal/metaserver/core/metadata"
	"github.com/22827099/DFS_v1/internal/metaserver/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectoryOperations(t *testing.T) {
	t.Run("CreateDirectoryTest", func(t *testing.T) {
		// 创建存储实例
		store, err := server.NewMemoryStore()
		require.NoError(t, err)
		require.NoError(t, store.Initialize())

		// 创建目录
		dirInfo := metadata.DirectoryInfo{
			Path: "/test_dir",
			Name: "test_dir",
		}
		result, err := store.CreateDirectory(context.Background(), dirInfo)
		require.NoError(t, err)
		assert.Equal(t, dirInfo.Path+"/", result.Path) // 添加了斜杠
		assert.Equal(t, dirInfo.Name, result.Name)

		// 测试创建嵌套目录
		nestedDir := metadata.DirectoryInfo{
			Path: "/test_dir/nested",
			Name: "nested",
		}
		result, err = store.CreateDirectory(context.Background(), nestedDir)
		require.NoError(t, err)
		assert.Equal(t, nestedDir.Path+"/", result.Path)
		assert.Equal(t, nestedDir.Name, result.Name)

		// 测试创建已存在的目录
		_, err = store.CreateDirectory(context.Background(), dirInfo)
		assert.Error(t, err)

		// 测试创建父目录不存在的目录
		invalidDir := metadata.DirectoryInfo{
			Path: "/non_existent/subdir",
			Name: "subdir",
		}
		_, err = store.CreateDirectory(context.Background(), invalidDir)
		assert.Error(t, err)
	})

	t.Run("DeleteDirectoryTest", func(t *testing.T) {
		// 创建存储实例
		store, err := server.NewMemoryStore()
		require.NoError(t, err)
		require.NoError(t, store.Initialize())

		// 创建目录
		dirInfo := metadata.DirectoryInfo{
			Path: "/dir_to_delete",
			Name: "dir_to_delete",
		}
		_, err = store.CreateDirectory(context.Background(), dirInfo)
		require.NoError(t, err)

		// 删除目录
		err = store.DeleteDirectory(context.Background(), "/dir_to_delete", false)
		require.NoError(t, err)

		// 验证目录已删除
		entries, err := store.ListDirectory(context.Background(), "/", true, 0)
		require.NoError(t, err)
		for _, entry := range entries {
			assert.NotEqual(t, "/dir_to_delete", entry.Path)
		}

		// 测试递归删除
		// 创建有嵌套内容的目录
		parentDir := metadata.DirectoryInfo{
			Path: "/parent_dir",
			Name: "parent_dir",
		}
		_, err = store.CreateDirectory(context.Background(), parentDir)
		require.NoError(t, err)

		childDir := metadata.DirectoryInfo{
			Path: "/parent_dir/child_dir",
			Name: "child_dir",
		}
		_, err = store.CreateDirectory(context.Background(), childDir)
		require.NoError(t, err)

		fileInfo := metadata.FileInfo{
			Path: "/parent_dir/test.txt",
			Name: "test.txt",
			Size: 1024,
		}
		_, err = store.CreateFile(context.Background(), fileInfo)
		require.NoError(t, err)

		// 测试非递归删除，应当失败
		err = store.DeleteDirectory(context.Background(), "/parent_dir", false)
		assert.Error(t, err)

		// 测试递归删除
		err = store.DeleteDirectory(context.Background(), "/parent_dir", true)
		require.NoError(t, err)

		// 验证目录及内容都已删除
		entries, err = store.ListDirectory(context.Background(), "/", true, 0)
		require.NoError(t, err)
		for _, entry := range entries {
			assert.NotEqual(t, "/parent_dir", entry.Path)
			assert.NotEqual(t, "/parent_dir/child_dir", entry.Path)
			assert.NotEqual(t, "/parent_dir/test.txt", entry.Path)
		}

		// 测试删除不存在的目录
		err = store.DeleteDirectory(context.Background(), "/non_existent", false)
		assert.Error(t, err)

		// 测试删除根目录，应当失败
		err = store.DeleteDirectory(context.Background(), "/", false)
		assert.Error(t, err)
	})
}
