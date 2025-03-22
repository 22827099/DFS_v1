package namespace_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/22827099/DFS_v1/common/logging"
	"github.com/22827099/DFS_v1/internal/metaserver/core/metadata/lock"
	"github.com/22827099/DFS_v1/internal/metaserver/core/metadata/namespace"
	"github.com/22827099/DFS_v1/internal/metaserver/core/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository 是Repository接口的模拟实现
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) FindOne(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	callArgs := []interface{}{ctx, dest, query}
	for _, arg := range args {
		callArgs = append(callArgs, arg)
	}
	return m.Called(callArgs...).Error(0)
}

func (m *MockRepository) Find(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	callArgs := []interface{}{ctx, dest, query}
	for _, arg := range args {
		callArgs = append(callArgs, arg)
	}
	return m.Called(callArgs...).Error(0)
}

func (m *MockRepository) FindByID(ctx context.Context, id int64, dest interface{}) error {
	return m.Called(ctx, id, dest).Error(0)
}

func (m *MockRepository) Create(ctx context.Context, tx *sql.Tx, entity interface{}) (sql.Result, error) {
	args := m.Called(ctx, tx, entity)
	return args.Get(0).(sql.Result), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, tx *sql.Tx, entity interface{}) (sql.Result, error) {
	args := m.Called(ctx, tx, entity)
	return args.Get(0).(sql.Result), args.Error(1)
}

func (m *MockRepository) Delete(ctx context.Context, tx *sql.Tx, id int64) (sql.Result, error) {
	args := m.Called(ctx, tx, id)
	return args.Get(0).(sql.Result), args.Error(1)
}

// MockDirectoryRepository 是DirectoryRepository接口的模拟实现
type MockDirectoryRepository struct {
	MockRepository
}

func (m *MockDirectoryRepository) FindByParentAndName(ctx context.Context, parentID int64, name string, dest *models.DirectoryMetadata) error {
	return m.Called(ctx, parentID, name, dest).Error(0)
}

func (m *MockDirectoryRepository) FindChildren(ctx context.Context, dirID int64) ([]models.DirectoryMetadata, error) {
	args := m.Called(ctx, dirID)
	return args.Get(0).([]models.DirectoryMetadata), args.Error(1)
}

// MockFileRepository 是FileRepository接口的模拟实现
type MockFileRepository struct {
	MockRepository
}

func (m *MockFileRepository) FindByDirAndName(ctx context.Context, dirID int64, name string, dest *models.FileMetadata) error {
	return m.Called(ctx, dirID, name, dest).Error(0)
}

func (m *MockFileRepository) FindByDir(ctx context.Context, dirID int64) ([]models.FileMetadata, error) {
	args := m.Called(ctx, dirID)
	return args.Get(0).([]models.FileMetadata), args.Error(1)
}

// MockSQLResult 是sql.Result接口的模拟实现
type MockSQLResult struct {
	mock.Mock
}

func (m *MockSQLResult) LastInsertId() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSQLResult) RowsAffected() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// MockDBManager 是database.Manager的模拟实现
type MockDBManager struct {
	mock.Mock
}

// MockLockManager 是lock.Manager的模拟实现
type MockLockManager struct {
	mock.Mock
}

func (m *MockLockManager) AcquireLock(ctx context.Context, resourceID string) (lock.LockHandle, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).(lock.LockHandle), args.Error(1)
}

func (m *MockLockManager) ReleaseLock(ctx context.Context, handle lock.LockHandle) error {
	return m.Called(ctx, handle).Error(0)
}

// MockLockHandle 是lock.LockHandle的模拟实现
type MockLockHandle struct {
	ID string
}

// TestNamespaceManager 测试命名空间管理器
func TestNamespaceManager(t *testing.T) {
	// 创建测试上下文
	ctx := context.Background()

	t.Run("Start", func(t *testing.T) {
		// 创建Mock对象
		mockDirRepo := new(MockDirectoryRepository)
		mockFileRepo := new(MockFileRepository)
		mockLockMgr := new(MockLockManager)
		mockDB := new(MockDBManager)
		logger := logging.NewLogger()

		// 预期行为 - 根目录查询
		rootDir := struct {
			DirID int64 `db:"dir_id"`
		}{DirID: 1}
		mockDirRepo.On("FindOne", mock.Anything, mock.Anything,
			"parent_id IS NULL AND name='/'").Run(func(args mock.Arguments) {
			dest := args.Get(1).(*struct{ DirID int64 })
			*dest = rootDir
		}).Return(nil)

		// 创建namespace管理器
		manager, err := namespace.NewManager(mockDB, mockLockMgr, logger)
		require.NoError(t, err)

		// 设置Manager的储存库
		manager.SetRepositories(mockDirRepo, mockFileRepo)

		// 启动管理器
		err = manager.Start()
		require.NoError(t, err)

		// 验证调用
		mockDirRepo.AssertExpectations(t)
	})

	t.Run("ResolvePath_Root", func(t *testing.T) {
		// 创建Mock对象
		mockDirRepo := new(MockDirectoryRepository)
		mockFileRepo := new(MockFileRepository)
		mockLockMgr := new(MockLockManager)
		mockDB := new(MockDBManager)
		logger := logging.NewLogger()

		// 创建namespace管理器
		manager, err := namespace.NewManager(mockDB, mockLockMgr, logger)
		require.NoError(t, err)
		manager.SetRepositories(mockDirRepo, mockFileRepo)

		// 设置根目录缓存（模拟Start方法已执行）
		rootDirID := int64(1)
		manager.SetRootDirID(rootDirID)

		// 预期行为 - 查询根目录
		rootDir := models.DirectoryMetadata{
			DirID:      rootDirID,
			Name:       "/",
			Path:       "/",
			ParentID:   nil,
			CreatedAt:  time.Now(),
			ModifiedAt: time.Now(),
		}
		mockDirRepo.On("FindByID", ctx, rootDirID, mock.Anything).Run(func(args mock.Arguments) {
			dest := args.Get(2).(*models.DirectoryMetadata)
			*dest = rootDir
		}).Return(nil)

		// 测试解析根路径
		pathInfo, err := manager.ResolvePath(ctx, "/")
		require.NoError(t, err)
		assert.NotNil(t, pathInfo)
		assert.True(t, pathInfo.Exists)
		assert.True(t, pathInfo.IsDir)
		assert.Equal(t, "/", pathInfo.Path)

		// 验证调用
		mockDirRepo.AssertExpectations(t)
	})

	t.Run("ResolvePath_DeepPath", func(t *testing.T) {
		// 创建Mock对象
		mockDirRepo := new(MockDirectoryRepository)
		mockFileRepo := new(MockFileRepository)
		mockLockMgr := new(MockLockManager)
		mockDB := new(MockDBManager)
		logger := logging.NewLogger()

		// 创建namespace管理器
		manager, err := namespace.NewManager(mockDB, mockLockMgr, logger)
		require.NoError(t, err)
		manager.SetRepositories(mockDirRepo, mockFileRepo)

		// 设置根目录缓存（模拟Start方法已执行）
		rootDirID := int64(1)
		manager.SetRootDirID(rootDirID)

		// 预期行为 - 查询整个路径
		rootDir := models.DirectoryMetadata{
			DirID:      rootDirID,
			Name:       "/",
			Path:       "/",
			ParentID:   nil,
			CreatedAt:  time.Now(),
			ModifiedAt: time.Now(),
		}

		dir1 := models.DirectoryMetadata{
			DirID:      2,
			Name:       "dir1",
			Path:       "/dir1",
			ParentID:   &rootDirID,
			CreatedAt:  time.Now(),
			ModifiedAt: time.Now(),
		}

		file1 := models.FileMetadata{
			FileID:      10,
			Name:        "file.txt",
			Size:        1024,
			ParentDirID: dir1.DirID,
			CreatedAt:   time.Now(),
			ModifiedAt:  time.Now(),
		}

		// 设置根目录查询行为
		mockDirRepo.On("FindByID", ctx, rootDirID, mock.Anything).Run(func(args mock.Arguments) {
			dest := args.Get(2).(*models.DirectoryMetadata)
			*dest = rootDir
		}).Return(nil)

		// dir1目录查询行为
		mockDirRepo.On("FindOne", ctx, mock.Anything,
			"parent_id = ? AND name = ? AND is_deleted = false", rootDirID, "dir1").
			Run(func(args mock.Arguments) {
				dest := args.Get(1).(*models.DirectoryMetadata)
				*dest = dir1
			}).Return(nil)

		// file.txt文件查询行为
		mockFileRepo.On("FindOne", ctx, mock.Anything,
			"parent_dir_id = ? AND name = ? AND is_deleted = false", dir1.DirID, "file.txt").
			Run(func(args mock.Arguments) {
				dest := args.Get(1).(*models.FileMetadata)
				*dest = file1
			}).Return(nil)

		// 测试解析路径
		pathInfo, err := manager.ResolvePath(ctx, "/dir1/file.txt")
		require.NoError(t, err)
		assert.NotNil(t, pathInfo)
		assert.True(t, pathInfo.Exists)
		assert.True(t, pathInfo.IsFile)
		assert.False(t, pathInfo.IsDir)
		assert.Equal(t, "/dir1/file.txt", pathInfo.Path)
		assert.Equal(t, "file.txt", pathInfo.Name)
		assert.Equal(t, "/dir1", pathInfo.ParentPath)

		// 验证调用
		mockDirRepo.AssertExpectations(t)
		mockFileRepo.AssertExpectations(t)
	})

	t.Run("ResolvePath_NonExistentPath", func(t *testing.T) {
		// 创建Mock对象
		mockDirRepo := new(MockDirectoryRepository)
		mockFileRepo := new(MockFileRepository)
		mockLockMgr := new(MockLockManager)
		mockDB := new(MockDBManager)
		logger := logging.NewLogger()

		// 创建namespace管理器
		manager, err := namespace.NewManager(mockDB, mockLockMgr, logger)
		require.NoError(t, err)
		manager.SetRepositories(mockDirRepo, mockFileRepo)

		// 设置根目录缓存（模拟Start方法已执行）
		rootDirID := int64(1)
		manager.SetRootDirID(rootDirID)

		// 预期行为 - 查询目录
		rootDir := models.DirectoryMetadata{
			DirID:      rootDirID,
			Name:       "/",
			Path:       "/",
			ParentID:   nil,
			CreatedAt:  time.Now(),
			ModifiedAt: time.Now(),
		}

		// 设置根目录查询行为
		mockDirRepo.On("FindByID", ctx, rootDirID, mock.Anything).Run(func(args mock.Arguments) {
			dest := args.Get(2).(*models.DirectoryMetadata)
			*dest = rootDir
		}).Return(nil)

		// 设置不存在目录的查询行为
		mockDirRepo.On("FindOne", ctx, mock.Anything,
			"parent_id = ? AND name = ? AND is_deleted = false", rootDirID, "nonexistent").
			Return(errors.New("directory not found"))

		mockFileRepo.On("FindOne", ctx, mock.Anything,
			"parent_dir_id = ? AND name = ? AND is_deleted = false", rootDirID, "nonexistent").
			Return(errors.New("file not found"))

		// 测试解析路径
		pathInfo, err := manager.ResolvePath(ctx, "/nonexistent")
		require.NoError(t, err)
		assert.NotNil(t, pathInfo)
		assert.False(t, pathInfo.Exists)
		assert.Equal(t, "/nonexistent", pathInfo.Path)
		assert.Equal(t, "nonexistent", pathInfo.Name)
		assert.Equal(t, "/", pathInfo.ParentPath)

		// 验证调用
		mockDirRepo.AssertExpectations(t)
		mockFileRepo.AssertExpectations(t)
	})

	t.Run("ListDirectory", func(t *testing.T) {
		// 创建Mock对象
		mockDirRepo := new(MockDirectoryRepository)
		mockFileRepo := new(MockFileRepository)
		mockLockMgr := new(MockLockManager)
		mockDB := new(MockDBManager)
		logger := logging.NewLogger()

		// 创建namespace管理器
		manager, err := namespace.NewManager(mockDB, mockLockMgr, logger)
		require.NoError(t, err)
		manager.SetRepositories(mockDirRepo, mockFileRepo)

		// 设置根目录缓存（模拟Start方法已执行）
		rootDirID := int64(1)
		manager.SetRootDirID(rootDirID)

		// 预期行为 - 目录列表查询
		rootDir := models.DirectoryMetadata{
			DirID:      rootDirID,
			Name:       "/",
			Path:       "/",
			ParentID:   nil,
			CreatedAt:  time.Now(),
			ModifiedAt: time.Now(),
		}

		// 模拟子目录数据
		childDirs := []models.DirectoryMetadata{
			{
				DirID:      2,
				Name:       "dir1",
				Path:       "/dir1",
				ParentID:   &rootDirID,
				CreatedAt:  time.Now(),
				ModifiedAt: time.Now(),
			},
			{
				DirID:      3,
				Name:       "dir2",
				Path:       "/dir2",
				ParentID:   &rootDirID,
				CreatedAt:  time.Now(),
				ModifiedAt: time.Now(),
			},
		}

		// 模拟子文件数据
		childFiles := []models.FileMetadata{
			{
				FileID:      10,
				Name:        "file1.txt",
				Size:        1024,
				ParentDirID: rootDirID,
				CreatedAt:   time.Now(),
				ModifiedAt:  time.Now(),
			},
			{
				FileID:      11,
				Name:        "file2.txt",
				Size:        2048,
				ParentDirID: rootDirID,
				CreatedAt:   time.Now(),
				ModifiedAt:  time.Now(),
			},
		}

		// 设置根目录查询行为
		mockDirRepo.On("FindByID", ctx, rootDirID, mock.Anything).Run(func(args mock.Arguments) {
			dest := args.Get(2).(*models.DirectoryMetadata)
			*dest = rootDir
		}).Return(nil)

		// 设置目录子项查询行为
		mockDirRepo.On("FindAll", ctx, mock.Anything,
			"parent_id = ? AND is_deleted = false", rootDirID).
			Run(func(args mock.Arguments) {
				dest := args.Get(1).(*[]models.DirectoryMetadata)
				*dest = childDirs
			}).Return(nil)

		// 设置文件子项查询行为
		mockFileRepo.On("FindAll", ctx, mock.Anything,
			"parent_dir_id = ? AND is_deleted = false", rootDirID).
			Run(func(args mock.Arguments) {
				dest := args.Get(1).(*[]models.FileMetadata)
				*dest = childFiles
			}).Return(nil)

		// 测试列出目录内容
		items, err := manager.ListDirectory(ctx, "/")
		require.NoError(t, err)
		assert.NotNil(t, items)
		assert.Equal(t, 4, len(items)) // 2个目录 + 2个文件

		// 验证调用
		mockDirRepo.AssertExpectations(t)
		mockFileRepo.AssertExpectations(t)

		// 检查排序
		assert.Equal(t, "dir1", items[0].Name) // 目录应该排在前面
		assert.Equal(t, "dir2", items[1].Name)
		assert.Equal(t, "file1.txt", items[2].Name)
		assert.Equal(t, "file2.txt", items[3].Name)
	})

	t.Run("ListDirectory_WithSort", func(t *testing.T) {
		// 创建Mock对象
		mockDirRepo := new(MockDirectoryRepository)
		mockFileRepo := new(MockFileRepository)
		mockLockMgr := new(MockLockManager)
		mockDB := new(MockDBManager)
		logger := logging.NewLogger()

		// 创建namespace管理器
		manager, err := namespace.NewManager(mockDB, mockLockMgr, logger)
		require.NoError(t, err)
		manager.SetRepositories(mockDirRepo, mockFileRepo)

		// 设置根目录缓存（模拟Start方法已执行）
		rootDirID := int64(1)
		manager.SetRootDirID(rootDirID)

		// 与上一个测试相同的数据设置
		// ...

		// 测试带排序的目录列表
		items, err := manager.ListDirectory(ctx, "/", namespace.WithSort("name", "desc"))
		require.NoError(t, err)
		// 验证排序效果，这里需要实际根据排序选项实现排序
		// ...
	})

	t.Run("Stop", func(t *testing.T) {
		// 创建Mock对象
		mockDirRepo := new(MockDirectoryRepository)
		mockFileRepo := new(MockFileRepository)
		mockLockMgr := new(MockLockManager)
		mockDB := new(MockDBManager)
		logger := logging.NewLogger()

		// 创建namespace管理器
		manager, err := namespace.NewManager(mockDB, mockLockMgr, logger)
		require.NoError(t, err)
		manager.SetRepositories(mockDirRepo, mockFileRepo)

		// 测试停止管理器
		err = manager.Stop(ctx)
		require.NoError(t, err)
	})
}
