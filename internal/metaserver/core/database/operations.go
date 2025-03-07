package database

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
)

var (
	// ErrNoRows 没有找到记录
	ErrNoRows = sql.ErrNoRows
	// ErrNotImplemented 功能未实现
	ErrNotImplemented = errors.New("功能未实现")
	// ErrInvalidArgument 无效的参数
	ErrInvalidArgument = errors.New("无效的参数")
)

// Transaction 表示一个数据库事务操作
type Transaction struct {
	tx  *sql.Tx
	mgr *Manager
}

// NewTransaction 从事务创建一个Transaction对象
func NewTransaction(tx *sql.Tx, mgr *Manager) *Transaction {
	return &Transaction{
		tx:  tx,
		mgr: mgr,
	}
}

// Commit 提交事务
func (t *Transaction) Commit() error {
	if t.tx == nil {
		return errors.New("事务未初始化")
	}
	return t.tx.Commit()
}

// Rollback 回滚事务
func (t *Transaction) Rollback() error {
	if t.tx == nil {
		return errors.New("事务未初始化")
	}
	return t.tx.Rollback()
}

// Exec 在事务中执行SQL语句
func (t *Transaction) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if t.tx == nil {
		return nil, errors.New("事务未初始化")
	}
	return t.tx.ExecContext(ctx, query, args...)
}

// Query 在事务中执行查询
func (t *Transaction) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if t.tx == nil {
		return nil, errors.New("事务未初始化")
	}
	return t.tx.QueryContext(ctx, query, args...)
}

// QueryRow 在事务中执行单行查询
func (t *Transaction) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if t.tx == nil {
		return nil
	}
	return t.tx.QueryRowContext(ctx, query, args...)
}

// DoInTransaction 在事务中执行函数
func (m *Manager) DoInTransaction(ctx context.Context, fn func(*Transaction) error) error {
	tx, err := m.GetTx(ctx)
	if err != nil {
		return err
	}

	txWrap := NewTransaction(tx, m)

	// 如果函数执行出错，回滚事务
	if err := fn(txWrap); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			m.logger.Error("回滚事务失败: %v (原始错误: %v)", rbErr, err)
		}
		return err
	}

	// 提交事务
	return tx.Commit()
}

// ScanRows 将行扫描到结构体切片中
func ScanRows(rows *sql.Rows, dest interface{}) error {
	// 检查dest是否是指向切片的指针
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return errors.New("目标必须是指向切片的指针")
	}

	sliceVal := destValue.Elem()
	elemType := sliceVal.Type().Elem()

	// 获取列信息
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	// 处理每一行
	for rows.Next() {
		// 创建新的元素
		newElem := reflect.New(elemType).Elem()

		// 为每个列创建扫描目标
		scanTargets := make([]interface{}, len(columns))
		for i, colName := range columns {
			field := newElem.FieldByName(colName)
			if !field.IsValid() {
				continue // 忽略结构体中不存在的字段
			}
			scanTargets[i] = field.Addr().Interface()
		}

		// 扫描行到目标
		if err := rows.Scan(scanTargets...); err != nil {
			return err
		}

		// 添加到结果切片
		sliceVal = reflect.Append(sliceVal, newElem)
	}

	// 检查遍历过程中的错误
	if err := rows.Err(); err != nil {
		return err
	}

	// 更新目标切片
	destValue.Elem().Set(sliceVal)
	return nil
}
