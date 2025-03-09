# 事务处理

此目录实现元数据的事务处理：

## 功能特性
- 事务隔离级别
  - Read Committed (读已提交)
  - Repeatable Read (可重复读)
  - Serializable (可串行化)
- 提交与回滚
  - 原子性保证
  - 提交时的一致性检查
  - 事务回滚机制
- 事务日志
  - 预写日志 (WAL)
  - 检查点机制
  - 日志恢复
- 分布式事务支持
  - 两阶段提交协议 (2PC)
  - 事务协调器
  - 分布式死锁检测

## 事务管理
事务管理器负责创建、提交和回滚事务，同时保证ACID属性：
- 原子性 (Atomicity)
- 一致性 (Consistency)
- 隔离性 (Isolation)
- 持久性 (Durability)

## 事务实现架构
1. 事务管理器 (TransactionManager)
2. 事务会话 (TransactionSession)
3. 事务日志管理器 (LogManager)
4. 锁管理器 (LockManager)
5. 分布式协调器 (CoordinatorService)

## 与其他模块的集成
- 元数据操作的事务包装
- 与存储引擎的集成
- 恢复机制
