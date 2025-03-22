package namespace

import (
	"context"
	"database/sql"

	"github.com/22827099/DFS_v1/internal/metaserver/core/models"
)

// Repository 定义了通用的数据访问接口
type Repository interface {
	// 基础查询方法
	FindOne(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Find(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	// 扩展查询方法 - 用于支持测试
	FindAll(ctx context.Context, dest interface{}, query string, args ...interface{}) error

	// 特定ID查询
	FindByID(ctx context.Context, id int64, dest interface{}) error

	// 创建、更新和删除操作
	Create(ctx context.Context, tx *sql.Tx, entity interface{}) (sql.Result, error)
	Update(ctx context.Context, tx *sql.Tx, entity interface{}) (sql.Result, error)
	Delete(ctx context.Context, tx *sql.Tx, id int64) (sql.Result, error)
}

// DirectoryRepository 定义了目录特有的数据访问接口
type DirectoryRepository interface {
	Repository
	FindByParentAndName(ctx context.Context, parentID int64, name string, dest *models.DirectoryMetadata) error
	FindChildren(ctx context.Context, dirID int64) ([]models.DirectoryMetadata, error)
}

// FileRepository 定义了文件特有的数据访问接口
type FileRepository interface {
	Repository
	FindByDirAndName(ctx context.Context, dirID int64, name string, dest *models.FileMetadata) error
	FindByDir(ctx context.Context, dirID int64) ([]models.FileMetadata, error)
}
