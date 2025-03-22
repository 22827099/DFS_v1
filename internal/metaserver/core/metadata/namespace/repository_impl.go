package namespace

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/22827099/DFS_v1/internal/metaserver/core/database"
	"github.com/22827099/DFS_v1/internal/metaserver/core/models"
)

// DirectoryRepositoryImpl 目录仓库实现
type DirectoryRepositoryImpl struct {
	baseRepo *database.Repository
	db       *database.Manager
	table    string
}

// FileRepositoryImpl 文件仓库实现
type FileRepositoryImpl struct {
	baseRepo *database.Repository
	db       *database.Manager
	table    string
}

// ========== 构造函数 ==========

// NewDirectoryRepository 创建目录仓库实现
func NewDirectoryRepository(db *database.Manager) DirectoryRepository {
	return &DirectoryRepositoryImpl{
		baseRepo: database.NewRepository(db, "directories"),
		db:       db,
		table:    "directories",
	}
}

// NewFileRepository 创建文件仓库实现
func NewFileRepository(db *database.Manager) FileRepository {
	return &FileRepositoryImpl{
		baseRepo: database.NewRepository(db, "files"),
		db:       db,
		table:    "files",
	}
}

// ========== DirectoryRepositoryImpl 方法实现 ==========

// FindOne 查找单一记录
func (r *DirectoryRepositoryImpl) FindOne(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return r.baseRepo.FindOne(ctx, dest, query, args...)
}

// Find 查找多条记录
// 使用 FindOne 的实现，因为 baseRepo 没有 Find 方法
func (r *DirectoryRepositoryImpl) Find(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	// 假设我们需要自己实现 Find 逻辑
	qb := database.NewQueryBuilder(r.table).Where(query)
	for _, arg := range args {
		qb.AddArg(arg)
	}
	sql, queryArgs := qb.BuildSelect()

	rows, err := r.db.QueryContext(ctx, sql, queryArgs...)
	if err != nil {
		return fmt.Errorf("查询失败: %w", err)
	}
	defer rows.Close()

	// 这里需要根据实际情况实现扫描逻辑
	// 简化示例，实际情况可能需要使用反射处理各种类型
	return scanRows(rows, dest)
}

// 辅助函数：将数据库行扫描到结构体切片
func scanRows(rows *sql.Rows, dest interface{}) error {
	// 实际实现中应该使用反射处理不同类型
	// 此处仅为占位符
	return nil
}

// FindAll 查找所有记录 (为测试提供)
func (r *DirectoryRepositoryImpl) FindAll(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return r.Find(ctx, dest, query, args...)
}

// FindByID 通过ID查找目录
func (r *DirectoryRepositoryImpl) FindByID(ctx context.Context, id int64, dest interface{}) error {
	return r.baseRepo.FindOne(ctx, dest, "dir_id = ?", id)
}

// Create 创建目录
func (r *DirectoryRepositoryImpl) Create(ctx context.Context, tx *sql.Tx, entity interface{}) (sql.Result, error) {
	dir, ok := entity.(*models.DirectoryMetadata)
	if !ok {
		return nil, fmt.Errorf("实体类型不是 DirectoryMetadata")
	}

	query := `INSERT INTO directories 
              (name, path, parent_id, owner, group_name, 
               permission_mode, deleted, created_time, modified_time) 
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var result sql.Result
	var err error

	args := []interface{}{
		dir.Name,
		dir.Path,
		dir.ParentID,
		dir.Owner,
		dir.Group,
		dir.Mode,         // 修正: 使用正确的权限字段名
		dir.Deleted,      // 修正: 使用正确的删除标记字段名
		dir.CreatedTime,  // 修正: 使用正确的创建时间字段名
		dir.ModifiedTime, // 修正: 使用正确的修改时间字段名
	}

	if tx != nil {
		result, err = tx.ExecContext(ctx, query, args...)
	} else {
		result, err = r.db.ExecContext(ctx, query, args...)
	}

	if err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	return result, nil
}

// Update 更新目录
func (r *DirectoryRepositoryImpl) Update(ctx context.Context, tx *sql.Tx, entity interface{}) (sql.Result, error) {
	dir, ok := entity.(*models.DirectoryMetadata)
	if !ok {
		return nil, fmt.Errorf("实体类型不是 DirectoryMetadata")
	}

	query := `UPDATE directories 
              SET name = ?, path = ?, parent_id = ?, owner = ?, group_name = ?, 
                  permission_mode = ?, deleted = ?, modified_time = ? 
              WHERE dir_id = ?`

	var result sql.Result
	var err error

	args := []interface{}{
		dir.Name,
		dir.Path,
		dir.ParentID,
		dir.Owner,
		dir.Group,
		dir.Mode,         // 修正: 使用正确的权限字段名
		dir.Deleted,      // 修正: 使用正确的删除标记字段名
		dir.ModifiedTime, // 修正: 使用正确的修改时间字段名
		dir.DirID,
	}

	if tx != nil {
		result, err = tx.ExecContext(ctx, query, args...)
	} else {
		result, err = r.db.ExecContext(ctx, query, args...)
	}

	if err != nil {
		return nil, fmt.Errorf("更新目录失败: %w", err)
	}

	return result, nil
}

// Delete 删除目录（逻辑删除）
func (r *DirectoryRepositoryImpl) Delete(ctx context.Context, tx *sql.Tx, id int64) (sql.Result, error) {
	query := `UPDATE directories SET deleted = true WHERE dir_id = ?`

	var result sql.Result
	var err error

	if tx != nil {
		result, err = tx.ExecContext(ctx, query, id)
	} else {
		result, err = r.db.ExecContext(ctx, query, id)
	}

	if err != nil {
		return nil, fmt.Errorf("删除目录失败: %w", err)
	}

	return result, nil
}

// FindByParentAndName 通过父ID和名称查找目录
func (r *DirectoryRepositoryImpl) FindByParentAndName(ctx context.Context, parentID int64, name string, dest *models.DirectoryMetadata) error {
	return r.baseRepo.FindOne(ctx, dest, "parent_id = ? AND name = ? AND deleted = false", parentID, name)
}

// FindChildren 查找子目录
func (r *DirectoryRepositoryImpl) FindChildren(ctx context.Context, dirID int64) ([]models.DirectoryMetadata, error) {
	var children []models.DirectoryMetadata
	// 由于 Find 需要自定义实现，我们直接使用 db.QueryContext
	query := "SELECT * FROM directories WHERE parent_id = ? AND deleted = false"
	rows, err := r.db.QueryContext(ctx, query, dirID)
	if err != nil {
		return nil, fmt.Errorf("查询子目录失败: %w", err)
	}
	defer rows.Close()

	// 手动扫描结果到切片
	for rows.Next() {
		var dir models.DirectoryMetadata
		// 这里应该根据实际的列结构来扫描
		err := rows.Scan(
			&dir.DirID,
			&dir.Name,
			&dir.Path,
			&dir.ParentID,
			&dir.Owner,
			&dir.Group,
			&dir.Mode,
			&dir.Deleted,
			&dir.CreatedTime,
			&dir.ModifiedTime,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描目录数据失败: %w", err)
		}
		children = append(children, dir)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历结果集失败: %w", err)
	}

	return children, nil
}

// ========== FileRepositoryImpl 方法实现 ==========

// FindOne 查找单一记录
func (r *FileRepositoryImpl) FindOne(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return r.baseRepo.FindOne(ctx, dest, query, args...)
}

// Find 查找多条记录
func (r *FileRepositoryImpl) Find(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	// 实现类似 DirectoryRepositoryImpl.Find 的逻辑
	qb := database.NewQueryBuilder(r.table).Where(query)
	for _, arg := range args {
		qb.AddArg(arg)
	}
	sql, queryArgs := qb.BuildSelect()

	rows, err := r.db.QueryContext(ctx, sql, queryArgs...)
	if err != nil {
		return fmt.Errorf("查询失败: %w", err)
	}
	defer rows.Close()

	return scanRows(rows, dest)
}

// FindAll 查找所有记录 (为测试提供)
func (r *FileRepositoryImpl) FindAll(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return r.Find(ctx, dest, query, args...)
}

// FindByID 通过ID查找文件
func (r *FileRepositoryImpl) FindByID(ctx context.Context, id int64, dest interface{}) error {
	return r.baseRepo.FindOne(ctx, dest, "file_id = ?", id)
}

// 序列化文件分片信息为JSON
func serializeChunks(file *models.FileMetadata) ([]byte, error) {
	return json.Marshal(file.Chunks)
}

// 从JSON反序列化文件分片信息
func deserializeChunks(file *models.FileMetadata, data []byte) error {
	return json.Unmarshal(data, &file.Chunks)
}

// Create 创建文件
func (r *FileRepositoryImpl) Create(ctx context.Context, tx *sql.Tx, entity interface{}) (sql.Result, error) {
	file, ok := entity.(*models.FileMetadata)
	if !ok {
		return nil, fmt.Errorf("实体类型不是 FileMetadata")
	}

	// 将分片信息序列化为JSON
	chunksJSON, err := serializeChunks(file)
	if err != nil {
		return nil, fmt.Errorf("序列化分片信息失败: %w", err)
	}

	query := `INSERT INTO files 
              (name, dir_id, size, chunks_data, mime_type, owner, group_name, 
               permission_mode, deleted, created_time, modified_time) 
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var result sql.Result

	args := []interface{}{
		file.Name,
		file.DirID, // 修正: 使用正确的目录ID字段名
		file.Size,
		chunksJSON,
		file.MimeType,
		file.Owner,
		file.Group,
		file.Mode,         // 修正: 使用正确的权限字段名
		file.Deleted,      // 修正: 使用正确的删除标记字段名
		file.CreatedTime,  // 修正: 使用正确的创建时间字段名
		file.ModifiedTime, // 修正: 使用正确的修改时间字段名
	}

	if tx != nil {
		result, err = tx.ExecContext(ctx, query, args...)
	} else {
		result, err = r.db.ExecContext(ctx, query, args...)
	}

	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %w", err)
	}

	return result, nil
}

// Update 更新文件
func (r *FileRepositoryImpl) Update(ctx context.Context, tx *sql.Tx, entity interface{}) (sql.Result, error) {
	file, ok := entity.(*models.FileMetadata)
	if !ok {
		return nil, fmt.Errorf("实体类型不是 FileMetadata")
	}

	// 将分片信息序列化为JSON
	chunksJSON, err := serializeChunks(file)
	if err != nil {
		return nil, fmt.Errorf("序列化分片信息失败: %w", err)
	}

	query := `UPDATE files 
              SET name = ?, dir_id = ?, size = ?, chunks_data = ?, mime_type = ?, 
                  owner = ?, group_name = ?, permission_mode = ?, deleted = ?, modified_time = ? 
              WHERE file_id = ?`

	var result sql.Result

	args := []interface{}{
		file.Name,
		file.DirID, // 修正: 使用正确的目录ID字段名
		file.Size,
		chunksJSON,
		file.MimeType,
		file.Owner,
		file.Group,
		file.Mode,         // 修正: 使用正确的权限字段名
		file.Deleted,      // 修正: 使用正确的删除标记字段名
		file.ModifiedTime, // 修正: 使用正确的修改时间字段名
		file.FileID,
	}

	if tx != nil {
		result, err = tx.ExecContext(ctx, query, args...)
	} else {
		result, err = r.db.ExecContext(ctx, query, args...)
	}

	if err != nil {
		return nil, fmt.Errorf("更新文件失败: %w", err)
	}

	return result, nil
}

// Delete 删除文件（逻辑删除）
func (r *FileRepositoryImpl) Delete(ctx context.Context, tx *sql.Tx, id int64) (sql.Result, error) {
	query := `UPDATE files SET deleted = true WHERE file_id = ?`

	var result sql.Result
	var err error

	if tx != nil {
		result, err = tx.ExecContext(ctx, query, id)
	} else {
		result, err = r.db.ExecContext(ctx, query, id)
	}

	if err != nil {
		return nil, fmt.Errorf("删除文件失败: %w", err)
	}

	return result, nil
}

// FindByDirAndName 通过目录ID和名称查找文件
func (r *FileRepositoryImpl) FindByDirAndName(ctx context.Context, dirID int64, name string, dest *models.FileMetadata) error {
	err := r.baseRepo.FindOne(ctx, dest, "dir_id = ? AND name = ? AND deleted = false", dirID, name)

	// 如果找到了文件，尝试解析分片信息
	if err == nil && dest != nil {
		// 假设数据库查询结果中有个 chunks_data 字段，已经被扫描到了一个未导出的 rawChunks 字段
		var chunksData []byte
		// 实际中可能需要单独查询，或者确保扫描到了这个字段
		// 这里简化处理
		if dest.RawChunksData != nil {
			chunksData = dest.RawChunksData
			err = deserializeChunks(dest, chunksData)
			if err != nil {
				return fmt.Errorf("解析文件分片信息失败: %w", err)
			}
		}
	}

	return err
}

// FindByDir 查找目录中的所有文件
func (r *FileRepositoryImpl) FindByDir(ctx context.Context, dirID int64) ([]models.FileMetadata, error) {
	var files []models.FileMetadata
	// 直接使用 QueryContext 而不是依赖 Find 方法
	query := "SELECT * FROM files WHERE dir_id = ? AND deleted = false"
	rows, err := r.db.QueryContext(ctx, query, dirID)
	if err != nil {
		return nil, fmt.Errorf("查询目录文件失败: %w", err)
	}
	defer rows.Close()

	// 手动扫描结果到切片
	for rows.Next() {
		var file models.FileMetadata
		var chunksData []byte
		// 这里应该根据实际的列结构来扫描
		err := rows.Scan(
			&file.FileID,
			&file.Name,
			&file.DirID,
			&file.Size,
			&chunksData, // 分片数据先扫描到变量
			&file.MimeType,
			&file.Owner,
			&file.Group,
			&file.Mode,
			&file.Deleted,
			&file.CreatedTime,
			&file.ModifiedTime,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描文件数据失败: %w", err)
		}

		// 反序列化分片信息
		if err := deserializeChunks(&file, chunksData); err != nil {
			return nil, fmt.Errorf("解析文件分片信息失败: %w", err)
		}

		files = append(files, file)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历结果集失败: %w", err)
	}

	return files, nil
}
