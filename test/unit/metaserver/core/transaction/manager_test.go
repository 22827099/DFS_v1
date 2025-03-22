package transaction_test

import (
	"context"
	"testing"
	"time"

	"github.com/22827099/DFS_v1/common/logging"
	"github.com/22827099/DFS_v1/internal/metaserver/core/database"
	"github.com/22827099/DFS_v1/internal/metaserver/core/metadata/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDBManager 模拟数据库管理器
type MockDBManager struct {
	mock.Mock
}

func (m *MockDBManager) Begin() (*database.Transaction, error) {
	args := m.Called()
	return args.Get(0).(*database.Transaction), args.Error(1)
}

func (m *MockDBManager) Commit(tx *database.Transaction) error {
	args := m.Called(tx)
	return args.Error(0)
}

func (m *MockDBManager) Rollback(tx *database.Transaction) error {
	args := m.Called(tx)
	return args.Error(0)
}

// 其他必要的Mock方法...

// TestTransactionManager 测试事务管理器
func TestTransactionManager(t *testing.T) {
	t.Run("BasicTransactionTest", func(t *testing.T) {
		// 创建Mock
		mockDB := new(MockDBManager)
		logger := logging.NewLogger()

		// 创建事务管理器
		manager, err := transaction.NewManager(mockDB, logger)
		require.NoError(t, err)

		// 启动事务管理器
		err = manager.Start()
		require.NoError(t, err)

		// 设置Mock行为
		mockTx := &database.Transaction{}
		mockDB.On("Begin").Return(mockTx, nil)
		mockDB.On("Commit", mockTx).Return(nil)
		mockDB.On("Rollback", mockTx).Return(nil)

		// 测试开始事务
		txID, err := manager.Begin(context.Background())
		require.NoError(t, err)
		assert.NotEmpty(t, txID)

		// 测试提交事务
		err = manager.Commit(context.Background(), txID)
		require.NoError(t, err)

		// 测试回滚事务
		txID, err = manager.Begin(context.Background())
		require.NoError(t, err)
		err = manager.Rollback(context.Background(), txID)
		require.NoError(t, err)

		// 停止事务管理器
		err = manager.Stop(context.Background())
		require.NoError(t, err)

		// 验证调用
		mockDB.AssertExpectations(t)
	})

	t.Run("TransactionTimeoutTest", func(t *testing.T) {
		// 创建Mock
		mockDB := new(MockDBManager)
		logger := logging.NewLogger()

		// 创建事务管理器，设置较短的超时时间
		manager, err := transaction.NewManager(mockDB, logger,
			transaction.WithTransactionTimeout(100*time.Millisecond))
		require.NoError(t, err)

		// 启动事务管理器
		err = manager.Start()
		require.NoError(t, err)

		// 设置Mock行为
		mockTx := &database.Transaction{}
		mockDB.On("Begin").Return(mockTx, nil)
		mockDB.On("Rollback", mockTx).Return(nil)

		// 开始事务
		txID, err := manager.Begin(context.Background())
		require.NoError(t, err)

		// 等待超时
		time.Sleep(200 * time.Millisecond)

		// 超时后尝试提交，应当失败
		err = manager.Commit(context.Background(), txID)
		assert.Error(t, err)

		// 停止事务管理器
		err = manager.Stop(context.Background())
		require.NoError(t, err)
	})
}
