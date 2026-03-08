package common

import (
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
)

// RedisClient Redis客户端封装
type RedisClient struct {
	pool *redis.Pool
}

// NewRedisClient 创建Redis客户端
func NewRedisClient(addr, password string, db, poolSize, timeout int) (*RedisClient, error) {
	pool := &redis.Pool{
		MaxIdle:     poolSize,
		IdleTimeout:  time.Duration(timeout) * time.Second,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp", addr)
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := conn.Do("AUTH", password); err != nil {
					conn.Close()
					return nil, err
				}
			}
			if db > 0 {
				if _, err := conn.Do("SELECT", db); err != nil {
					conn.Close()
					return nil, err
				}
			}
			return conn, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
	return &RedisClient{pool: pool}, nil
}

// Get 获取值
func (r *RedisClient) Get(key string) (string, error) {
	conn := r.pool.Get()
	defer conn.Close()
	return redis.String(conn.Do("GET", key))
}

// Set 设置值
func (r *RedisClient) Set(key string, value interface{}, ttl int) error {
	conn := r.pool.Get()
	defer conn.Close()
	if ttl > 0 {
		_, err := conn.Do("SETEX", key, ttl, value)
		return err
	}
	_, err := conn.Do("SET", key, value)
	return err
}

// Del 删除键
func (r *RedisClient) Del(keys ...string) error {
	conn := r.pool.Get()
	defer conn.Close()
	args := make([]interface{}, len(keys))
	for i, key := range keys {
		args[i] = key
	}
	_, err := conn.Do("DEL", args...)
	return err
}

// Incr 自增
func (r *RedisClient) Incr(key string) (int64, error) {
	conn := r.pool.Get()
	defer conn.Close()
	return redis.Int64(conn.Do("INCR", key))
}

// IncrBy 按指定值自增
func (r *RedisClient) IncrBy(key string, value int64) (int64, error) {
	conn := r.pool.Get()
	defer conn.Close()
	return redis.Int64(conn.Do("INCRBY", key, value))
}

// DecrBy 按指定值自减
func (r *RedisClient) DecrBy(key string, value int64) (int64, error) {
	conn := r.pool.Get()
	defer conn.Close()
	return redis.Int64(conn.Do("DECRBY", key, value))
}

// Expire 设置过期时间
func (r *RedisClient) Expire(key string, ttl int) error {
	conn := r.pool.Get()
	defer conn.Close()
	_, err := conn.Do("EXPIRE", key, ttl)
	return err
}

// Eval 执行Lua脚本
func (r *RedisClient) Eval(script string, keyCount int, keys []string, args []interface{}) (interface{}, error) {
	conn := r.pool.Get()
	defer conn.Close()

	// 准备所有参数：script, keyCount, keys..., args...
	allArgs := make([]interface{}, 0, 2+len(keys)+len(args))
	allArgs = append(allArgs, script)
	allArgs = append(allArgs, keyCount)
	for _, key := range keys {
		allArgs = append(allArgs, key)
	}
	allArgs = append(allArgs, args...)

	return conn.Do("EVAL", allArgs...)
}

// Close 关闭连接池
func (r *RedisClient) Close() error {
	return r.pool.Close()
}

// BuildRedisKey 构建Redis Key
func BuildRedisKey(prefix, service, consumer, suffix string) string {
	return fmt.Sprintf("%s:%s:%s:%s", prefix, service, consumer, suffix)
}

// BuildTokenLimitKey 构建Token限流Key
func BuildTokenLimitKey(service, consumer, window string) string {
	return fmt.Sprintf("ai_token:%s:%s:%s", service, consumer, window)
}

// BuildQuotaKey 构建配额Key
func BuildQuotaKey(consumer string) string {
	return fmt.Sprintf("ai_quota:%s", consumer)
}

// BuildCacheKey 构建缓存Key
func BuildCacheKey(question string) string {
	if question == "" {
		return "ai_cache:empty"
	}
	return fmt.Sprintf("ai_cache:question:%s", Md5(question))
}

// BuildLoadBalancerKey 构建负载均衡Key
func BuildLoadBalancerKey(node string) string {
	return fmt.Sprintf("ai_lb:ongoing:%s", node)
}
