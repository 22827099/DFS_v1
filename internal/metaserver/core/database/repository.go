package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Repository 提供类型安全的数据访问
type Repository struct {
	manager *Manager
	table   string
}

func (r *Repository) FindChildren(ctx context.Context, d int64) (any, error) {
	panic("unimplemented")
}

// NewRepository 创建新的存储库
func NewRepository(manager *Manager, table string) *Repository {
	return &Repository{
		manager: manager,
		table:   table,
	}
}

// FindByID 按ID查找记录
func (r *Repository) FindByID(ctx context.Context, id interface{}, dest interface{}) error {
	qb := NewQueryBuilder(r.table).Where("id = ?", id)
	query, args := qb.BuildSelect()

	row := r.manager.QueryRowContext(ctx, query, args...)
	return scanStruct(row, dest)
}

// FindOne 查找单个记录
func (r *Repository) FindOne(ctx context.Context, dest interface{}, where string, args ...interface{}) error {
	qb := NewQueryBuilder(r.table).Where(where)
	for _, arg := range args {
		qb.whereArgs = append(qb.whereArgs, arg)
	}
	query, queryArgs := qb.BuildSelect()

	row := r.manager.QueryRowContext(ctx, query, queryArgs...)
	return scanStruct(row, dest)
}

// FindAll 查找多个记录
func (r *Repository) FindAll(ctx context.Context, dest interface{}, where string, args ...interface{}) error {
	qb := NewQueryBuilder(r.table)
	if where != "" {
		qb.Where(where)
		for _, arg := range args {
			qb.whereArgs = append(qb.whereArgs, arg)
		}
	}
	query, queryArgs := qb.BuildSelect()

	rows, err := r.manager.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return err
	}
	defer rows.Close()

	return ScanRows(rows, dest)
}

// Insert 插入记录
func (r *Repository) Insert(ctx context.Context, entity interface{}) (sql.Result, error) {
	columns, values, err := extractInsertValues(entity)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		r.table,
		strings.Join(columns, ", "),
		strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "),
	)

	return r.manager.ExecContext(ctx, query, values...)
}

// Update 更新记录
func (r *Repository) Update(ctx context.Context, entity interface{}, where string, args ...interface{}) (sql.Result, error) {
	columns, values, err := extractInsertValues(entity)
	if err != nil {
		return nil, err
	}

	// 构建SET部分
	setParts := make([]string, len(columns))
	for i, col := range columns {
		setParts[i] = fmt.Sprintf("%s = ?", col)
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		r.table,
		strings.Join(setParts, ", "),
		where,
	)

	// 合并所有参数
	allArgs := append(values, args...)

	return r.manager.ExecContext(ctx, query, allArgs...)
}

// Delete 删除记录
func (r *Repository) Delete(ctx context.Context, where string, args ...interface{}) (sql.Result, error) {
	qb := NewQueryBuilder(r.table)
	if where != "" {
		qb.Where(where)
		for _, arg := range args {
			qb.whereArgs = append(qb.whereArgs, arg)
		}
	} else {
		return nil, errors.New("删除操作必须指定WHERE条件")
	}

	query, queryArgs := qb.BuildDelete()
	return r.manager.ExecContext(ctx, query, queryArgs...)
}

// Count 计算记录数
func (r *Repository) Count(ctx context.Context, where string, args ...interface{}) (int, error) {
	qb := NewQueryBuilder(r.table)
	if where != "" {
		qb.Where(where)
		for _, arg := range args {
			qb.whereArgs = append(qb.whereArgs, arg)
		}
	}
	query, queryArgs := qb.BuildCount()

	var count int
	err := r.manager.QueryRowContext(ctx, query, queryArgs...).Scan(&count)
	return count, err
}

// 辅助函数

// scanStruct 将行扫描到结构体
func scanStruct(row *sql.Row, dest interface{}) error {
	// 获取目标类型信息
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr || destVal.Elem().Kind() != reflect.Struct {
		return errors.New("目标必须是指向结构体的指针")
	}

	// 获取结构体值
	structVal := destVal.Elem()
	structType := structVal.Type()

	// 创建字段列表
	numFields := structType.NumField()
	fields := make([]interface{}, numFields)
	for i := 0; i < numFields; i++ {
		fields[i] = structVal.Field(i).Addr().Interface()
	}

	// 扫描行到字段
	return row.Scan(fields...)
}

// extractInsertValues 从结构体提取列和值
func extractInsertValues(entity interface{}) ([]string, []interface{}, error) {
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, nil, errors.New("实体必须是结构体或结构体指针")
	}

	typ := val.Type()
	columns := make([]string, 0, val.NumField())
	values := make([]interface{}, 0, val.NumField())

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)

		// 获取数据库列名
		colName := field.Tag.Get("db")
		if colName == "" {
			colName = strings.ToLower(field.Name)
		}

		// 忽略ID字段（假设是自增的）和设置为忽略的字段
		if colName == "-" || colName == "id" {
			continue
		}

		columns = append(columns, colName)
		values = append(values, val.Field(i).Interface())
	}

	return columns, values, nil
}
