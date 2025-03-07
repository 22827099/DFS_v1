package metadata

import (
	"context"
	"fmt"
	"sync"

	"github.com/22827099/DFS_v1/common/logging"
	"github.com/22827099/DFS_v1/internal/metaserver/core/database"
	"github.com/22827099/DFS_v1/internal/metaserver/core/metadata/lock"
	"github.com/22827099/DFS_v1/internal/metaserver/core/metadata/namespace"
	"github.com/22827099/DFS_v1/internal/metaserver/core/metadata/transaction"
)

// Manager 表示元数据管理器
type Manager struct {
	db          *database.Manager
	logger      logging.Logger
	lockMgr     *lock.Manager
	nsMgr       *namespace.Manager
	txMgr       *transaction.Manager
	fileRepo    *database.Repository
	dirRepo     *database.Repository
	chunkRepo   *database.Repository
	replicaRepo *database.Repository
	mu          sync.RWMutex
}

// NewManager 创建新的元数据管理器
func NewManager(db *database.Manager, logger logging.Logger) (*Manager, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库管理器不能为空")
	}

	lockMgr, err := lock.NewManager(logger)
	if err != nil {
		return nil, fmt.Errorf("初始化锁管理器失败: %w", err)
	}

	txMgr, err := transaction.NewManager(db, logger)
	if err != nil {
		return nil, fmt.Errorf("初始化事务管理器失败: %w", err)
	}

	nsMgr, err := namespace.NewManager(db, lockMgr, logger)
	if err != nil {
		return nil, fmt.Errorf("初始化命名空间管理器失败: %w", err)
	}

	// 创建各种仓库
	fileRepo := database.NewRepository(db, "files")
	dirRepo := database.NewRepository(db, "directories")
	chunkRepo := database.NewRepository(db, "chunks")
	replicaRepo := database.NewRepository(db, "replicas")

	return &Manager{
		db:          db,
		logger:      logger,
		lockMgr:     lockMgr,
		nsMgr:       nsMgr,
		txMgr:       txMgr,
		fileRepo:    fileRepo,
		dirRepo:     dirRepo,
		chunkRepo:   chunkRepo,
		replicaRepo: replicaRepo,
	}, nil
}

// Start 启动元数据管理器
func (m *Manager) Start() error {
	m.logger.Info("启动元数据管理器")

	// 启动各子管理器
	if err := m.lockMgr.Start(); err != nil {
		return fmt.Errorf("启动锁管理器失败: %w", err)
	}

	if err := m.txMgr.Start(); err != nil {
		return fmt.Errorf("启动事务管理器失败: %w", err)
	}

	if err := m.nsMgr.Start(); err != nil {
		return fmt.Errorf("启动命名空间管理器失败: %w", err)
	}

	return nil
}

// Stop 停止元数据管理器
func (m *Manager) Stop(ctx context.Context) error {
	m.logger.Info("停止元数据管理器")

	// 按照依赖关系的相反顺序关闭
	if err := m.nsMgr.Stop(ctx); err != nil {
		m.logger.Error("停止命名空间管理器失败: %v", err)
	}

	if err := m.txMgr.Stop(ctx); err != nil {
		m.logger.Error("停止事务管理器失败: %v", err)
	}

	if err := m.lockMgr.Stop(ctx); err != nil {
		m.logger.Error("停止锁管理器失败: %v", err)
	}

	return nil
}
