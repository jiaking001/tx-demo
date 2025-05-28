package pkg

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"time"
)

// LockConfig 锁的配置
type LockConfig struct {
	// 默认过期时间
	DefaultExpiration time.Duration
	// 默认等待时间
	DefaultWaitTime time.Duration
	// 锁的key前缀
	KeyPrefix string
	// 最大重试次数
	MaxRetries int
	// 重试间隔
	RetryInterval time.Duration
}

// DefaultLockConfig 默认配置
var DefaultLockConfig = LockConfig{
	DefaultExpiration: 10 * time.Second,
	DefaultWaitTime:   5 * time.Second,
	KeyPrefix:         "lock:",
	MaxRetries:        3,
	RetryInterval:     100 * time.Millisecond,
}

// RedisLock 分布式锁结构体
type RedisLock struct {
	rdb        *redis.Client
	logger     *zap.Logger
	key        string
	value      string
	expiration time.Duration
	config     LockConfig
	renewCh    chan struct{}
	stopRenew  chan struct{}
}

// NewRedisLock 创建一个新的分布式锁实例
func NewRedisLock(rdb *redis.Client, logger *zap.Logger, key string, config LockConfig) *RedisLock {
	return &RedisLock{
		rdb:        rdb,
		logger:     logger,
		key:        fmt.Sprintf("%s%s", config.KeyPrefix, key),
		value:      fmt.Sprintf("%d-%s", time.Now().UnixNano(), uuid.New().String()),
		expiration: config.DefaultExpiration,
		config:     config,
		// 需要续期时传入信号
		renewCh: make(chan struct{}),
		// 停止续期时传入信号
		stopRenew: make(chan struct{}),
	}
}

// Lock 获取锁
func (l *RedisLock) Lock(ctx context.Context) (bool, error) {
	startTime := time.Now()
	// 启动自动续期
	go l.autoRenew(ctx)

	// 尝试获取锁
	for i := 0; i < l.config.MaxRetries; i++ {
		// 使用SETNX命令尝试获取锁
		set, err := l.rdb.SetNX(ctx, l.key, l.value, l.expiration).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				// 键不存在，继续重试
				time.Sleep(l.config.RetryInterval)
				continue
			}
			return false, fmt.Errorf("failed to acquire lock: %w", err)
		}

		if set {
			return true, nil
		}

		// 检查是否超过等待时间
		if time.Since(startTime) > l.config.DefaultWaitTime {
			return false, nil
		}

		time.Sleep(l.config.RetryInterval)
	}

	return false, nil
}

// autoRenew 自动续期
func (l *RedisLock) autoRenew(ctx context.Context) {
	ticker := time.NewTicker(l.expiration / 3)
	defer ticker.Stop()

	for {
		select {
		// 自动续期
		case <-ticker.C:
			// 续期逻辑
			if err := l.renew(ctx); err != nil {
				// 续期失败，停止续期,并记录日志
				l.logger.Error("failed to renew lock", zap.String("key", l.key), zap.String("value", l.value), zap.Error(err))
				return
			}
		// 手动续期
		case <-l.renewCh:
			// 续期逻辑
			if err := l.renew(ctx); err != nil {
				// 续期失败，停止续期,并记录日志
				l.logger.Error("failed to renew lock", zap.String("key", l.key), zap.String("value", l.value), zap.Error(err))
				return
			}
		case <-ctx.Done():
			return
		case <-l.stopRenew:
			return
		}
	}
}

// renew 续期
func (l *RedisLock) renew(ctx context.Context) error {
	// 使用Lua脚本确保原子性
	luaScript := `
        if redis.call("GET", KEYS[1]) == ARGV[1] then
            return redis.call("EXPIRE", KEYS[1], ARGV[2])
        else
            return 0
        end
    `

	result, err := l.rdb.Eval(ctx, luaScript, []string{l.key}, l.value, l.expiration.Seconds()).Result()
	if err != nil {
		return fmt.Errorf("failed to renew lock: %w", err)
	}

	if result.(int64) == 0 {
		return fmt.Errorf("lock not found or value mismatch")
	}

	return nil
}

// Unlock 释放锁
func (l *RedisLock) Unlock(ctx context.Context) error {
	// 停止续期
	close(l.stopRenew)

	// 使用Lua脚本确保原子性
	luaScript := `
        if redis.call("GET", KEYS[1]) == ARGV[1] then
            return redis.call("DEL", KEYS[1])
        else
            return 0
        end
    `

	result, err := l.rdb.Eval(ctx, luaScript, []string{l.key}, l.value).Result()
	if err != nil {
		return fmt.Errorf("failed to unlock: %w", err)
	}

	if result.(int64) == 0 {
		return fmt.Errorf("lock not found or value mismatch")
	}

	return nil
}
