package database

import (
	"fmt"
	"strings"
)

// QueryBuilder 帮助构建SQL查询
type QueryBuilder struct {
	table     string
	columns   []string
	where     []string
	whereArgs []interface{}
	orderBy   string
	limit     int
	offset    int
}

// NewQueryBuilder 创建新的查询构建器
func NewQueryBuilder(table string) *QueryBuilder {
	return &QueryBuilder{
		table:   table,
		columns: []string{"*"},
		limit:   -1,
		offset:  -1,
	}
}

// Select 设置要查询的列
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	qb.columns = columns
	return qb
}

// Where 添加WHERE条件
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	qb.where = append(qb.where, condition)
	qb.whereArgs = append(qb.whereArgs, args...)
	return qb
}

// OrderBy 设置排序
func (qb *QueryBuilder) OrderBy(orderBy string) *QueryBuilder {
	qb.orderBy = orderBy
	return qb
}

// Limit 设置限制
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

// Offset 设置偏移
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset
	return qb
}

// BuildSelect 构建SELECT查询
func (qb *QueryBuilder) BuildSelect() (string, []interface{}) {
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(qb.columns, ", "), qb.table)

	if len(qb.where) > 0 {
		query += " WHERE " + strings.Join(qb.where, " AND ")
	}

	if qb.orderBy != "" {
		query += " ORDER BY " + qb.orderBy
	}

	if qb.limit >= 0 {
		query += fmt.Sprintf(" LIMIT %d", qb.limit)
	}

	if qb.offset >= 0 {
		query += fmt.Sprintf(" OFFSET %d", qb.offset)
	}

	return query, qb.whereArgs
}

// BuildCount 构建COUNT查询
func (qb *QueryBuilder) BuildCount() (string, []interface{}) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", qb.table)

	if len(qb.where) > 0 {
		query += " WHERE " + strings.Join(qb.where, " AND ")
	}

	return query, qb.whereArgs
}

// BuildDelete 构建DELETE查询
func (qb *QueryBuilder) BuildDelete() (string, []interface{}) {
	query := fmt.Sprintf("DELETE FROM %s", qb.table)

	if len(qb.where) > 0 {
		query += " WHERE " + strings.Join(qb.where, " AND ")
	}

	return query, qb.whereArgs
}
