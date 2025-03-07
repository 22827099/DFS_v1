package namespace

import (
    "context"
    "fmt"
    "path/filepath"
    "strings"
    "sync"

    "github.com/22827099/DFS_v1/common/logging"
    "github.com/22827099/DFS_v1/internal/metaserver/core/database"
    "github.com/22827099/DFS_v1/internal/metaserver/core/metadata/lock"
	"github.com/22827099/DFS_v1/internal/metaserver/core/models"

)

// Manager 负责命名空间管理
type Manager struct {
	db        *database.Manager
	lockMgr   *lock.Manager
	logger    logging.Logger
	dirRepo   *database.Repository
	fileRepo  *database.Repository
	rootCache sync.Map // 缓存根目录ID
}

// NewManager 创建新的命名空间管理器
func NewManager(db *database.Manager, lockMgr *lock.Manager, logger logging.Logger) (*Manager, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库管理器不能为空")
	}

	if lockMgr == nil {
		return nil, fmt.Errorf("锁管理器不能为空")
	}

	dirRepo := database.NewRepository(db, "directories")
	fileRepo := database.NewRepository(db, "files")

	return &Manager{
		db:       db,
		lockMgr:  lockMgr,
		logger:   logger,
		dirRepo:  dirRepo,
		fileRepo: fileRepo,
	}, nil
}

// Start 启动命名空间管理器
func (m *Manager) Start() error {
	m.logger.Info("启动命名空间管理器")
	// 预加载根目录ID
	ctx := context.Background()
	rootDir := struct {
		DirID int64 `db:"dir_id"`
	}{}

	err := m.dirRepo.FindOne(ctx, &rootDir, "parent_id IS NULL AND name='/'")
	if err != nil {
		return fmt.Errorf("查找根目录失败: %w", err)
	}

	m.rootCache.Store("/", rootDir.DirID)
	return nil
}

// Stop 停止命名空间管理器
func (m *Manager) Stop(ctx context.Context) error {
	m.logger.Info("停止命名空间管理器")
	// 清除缓存
	m.rootCache = sync.Map{}
	return nil
}

// ResolvePath 将路径解析为目录或文件ID
func (m *Manager) ResolvePath(ctx context.Context, path string) (*models.PathInfo, error) {
    // 标准化路径
    path = filepath.Clean("/" + strings.TrimPrefix(path, "/"))

    // 检查根目录
    if path == "/" {
        var rootID int64
        value, ok := m.rootCache.Load("/")
        if !ok {
            return nil, fmt.Errorf("根目录未初始化")
        }
        rootID = value.(int64)

        var rootDir models.DirectoryMetadata
        if err := m.dirRepo.FindByID(ctx, rootID, &rootDir); err != nil {
            return nil, fmt.Errorf("获取根目录失败: %w", err)
        }

        return &models.PathInfo{
            Path:       "/",
            Exists:     true,
            IsDir:      true,
            Metadata:   rootDir,
            ParentPath: "/",
            Name:       "/",
        }, nil
    }

    // 获取父路径和名称
    parentPath := filepath.Dir(path)
    name := filepath.Base(path)

    // 首先解析父目录
    parentInfo, err := m.ResolvePath(ctx, parentPath)
    if err != nil {
        return nil, err
    }

    if !parentInfo.Exists || !parentInfo.IsDir {
        return &models.PathInfo{
            Path:       path,
            Exists:     false,
            ParentPath: parentPath,
            Name:       name,
            // 如果父目录存在但不是目录，保留其元数据
            ParentDir:  parentInfo.Metadata.(*models.DirectoryMetadata),
        }, nil
    }

    // 获取父目录的目录元数据
    parentDir := parentInfo.Metadata.(*models.DirectoryMetadata)

    // 尝试查找文件
    var file models.FileMetadata
    err = m.fileRepo.FindOne(ctx, &file, "parent_dir_id = ? AND name = ? AND is_deleted = false",
        parentDir.DirID, name)
    if err == nil {
        return &models.PathInfo{
            Path:       path,
            Exists:     true,
            IsFile:     true,
            IsDir:      false,
            Metadata:   file,
            ParentPath: parentPath,
            ParentDir:  parentDir,
            Name:       name,
        }, nil
    }

    // 尝试查找目录
    var dir models.DirectoryMetadata
    err = m.dirRepo.FindOne(ctx, &dir, "parent_id = ? AND name = ? AND is_deleted = false",
        parentDir.DirID, name)
    if err == nil {
        return &models.PathInfo{
            Path:       path,
            Exists:     true,
            IsFile:     false,
            IsDir:      true,
            Metadata:   dir,
            ParentPath: parentPath,
            ParentDir:  parentDir,
            Name:       name,
        }, nil
    }

    // 路径不存在
    return &models.PathInfo{
        Path:       path,
        Exists:     false,
        ParentPath: parentPath,
        ParentDir:  parentDir,
        Name:       name,
    }, nil
}

// ListDirectory 列出目录内容
func (m *Manager) ListDirectory(ctx context.Context, path string) ([]models.PathInfo, error) {
    // 解析指定路径
    pathInfo, err := m.ResolvePath(ctx, path)
    if err != nil {
        return nil, err
    }

    if !pathInfo.Exists {
        return nil, fmt.Errorf("目录不存在: %s", path)
    }

    if !pathInfo.IsDir {
        return nil, fmt.Errorf("路径不是目录: %s", path)
    }

    // 获取目录元数据
    dirMeta, ok := pathInfo.Metadata.(models.DirectoryMetadata)
    if !ok {
        return nil, fmt.Errorf("无效的目录元数据")
    }

    // 获取子文件和子目录
    var result []models.PathInfo

    // 获取子目录
    var childDirs []models.DirectoryMetadata
    err = m.dirRepo.FindAll(ctx, &childDirs, "parent_id = ? AND is_deleted = false", dirMeta.DirID)
    if err != nil {
        return nil, fmt.Errorf("获取子目录失败: %w", err)
    }

    for _, dir := range childDirs {
        childPath := filepath.Join(path, dir.Name)
        result = append(result, models.PathInfo{
            Path:       childPath,
            Exists:     true,
            IsDir:      true,
            IsFile:     false,
            Metadata:   dir,
            ParentPath: path,
            Name:       dir.Name,
        })
    }

    // 获取子文件
    var childFiles []models.FileMetadata
    err = m.fileRepo.FindAll(ctx, &childFiles, "parent_dir_id = ? AND is_deleted = false", dirMeta.DirID)
    if err != nil {
        return nil, fmt.Errorf("获取子文件失败: %w", err)
    }

    for _, file := range childFiles {
        childPath := filepath.Join(path, file.Name)
        result = append(result, models.PathInfo{
            Path:       childPath,
            Exists:     true,
            IsDir:      false,
            IsFile:     true,
            Metadata:   file,
            ParentPath: path,
            Name:       file.Name,
        })
    }

    return result, nil
}
