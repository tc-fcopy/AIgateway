package cache

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"gateway/ai_gateway/common"
)

// StringCache 字符串匹配缓存
type StringCache struct {
	enable       bool
	cacheTTL     int64
	maxCacheSize int
	cacheStream  bool
	redisClient  *common.RedisClient
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Response   []byte    `json:"response"`   // 响应内容
	Timestamp  time.Time `json:"timestamp"`   // 缓存时间
	ExpireTime time.Time `json:"expire_time"` // 过期时间
	TokenCount int       `json:"token_count"` // Token数量
}

// NewStringCache 创建字符串缓存
func NewStringCache(redisClient *common.RedisClient) *StringCache {
	return &StringCache{
		enable:       true,
		cacheTTL:     3600, // 默认1小时
		maxCacheSize: 1000, // 默认1000条
		cacheStream:  false,
		redisClient:  redisClient,
	}
}

// GenerateCacheKey 生成缓存Key
func (s *StringCache) GenerateCacheKey(consumer, model, prompt string) string {
	// 将关键信息拼接
	key := fmt.Sprintf("%s:%s:%s", consumer, model, prompt)

	// 计算MD5
	hash := md5.Sum([]byte(key))

	return hex.EncodeToString(hash[:])
}

// GenerateCacheKeyFromRequest 从请求生成缓存Key
func (s *StringCache) GenerateCacheKeyFromRequest(consumer, model string, body []byte) (string, error) {
	// 解析请求体提取prompt
	prompt, err := s.extractPrompt(body)
	if err != nil {
		return "", err
	}

	return s.GenerateCacheKey(consumer, model, prompt), nil
}

// extractPrompt 从请求体中提取prompt
func (s *StringCache) extractPrompt(body []byte) (string, error) {
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		return "", err
	}

	// 尝试获取messages字段
	if messages, ok := req["messages"].([]interface{}); ok {
		// 提取最后一个用户消息
		for i := len(messages) - 1; i >= 0; i-- {
			if msg, ok := messages[i].(map[string]interface{}); ok {
				if role, ok := msg["role"].(string); ok && role == "user" {
					if content, ok := msg["content"].(string); ok {
						return content, nil
					}
				}
			}
		}
	}

	// 尝试获取prompt字段
	if prompt, ok := req["prompt"].(string); ok {
		return prompt, nil
	}

	// 尝与其他取input字段
	if input, ok := req["input"].(string); ok {
		return input, nil
	}

	// 使用整个请求体作为fallback
	return string(body), nil
}

// Get 从缓存获取
func (s *StringCache) Get(cacheKey string) (*CacheEntry, error) {
	if !s.enable {
		return nil, fmt.Errorf("cache is disabled")
	}

	if s.redisClient == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	// 使用Redis获取
	val, err := s.redisClient.Get(common.BuildCacheKey(cacheKey))
	if err != nil {
		return nil, err
	}

	// 解析缓存条目
	var entry CacheEntry
	if err := json.Unmarshal([]byte(val), &entry); err != nil {
		return nil, err
	}

	// 检查是否过期
	if time.Now().After(entry.ExpireTime) {
		// 删除过期缓存
		_ = s.redisClient.Del(common.BuildCacheKey(cacheKey))
		return nil, fmt.Errorf("cache entry expired")
	}

	return &entry, nil
}

// Set 设置缓存
func (s *StringCache) Set(cacheKey string, response []byte, tokenCount int) error {
	if !s.enable {
		return nil
	}

	if s.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	// 创建缓存条目
	entry := CacheEntry{
		Response:   response,
		Timestamp:  time.Now(),
		ExpireTime: time.Now().Add(time.Duration(s.cacheTTL) * time.Second),
		TokenCount: tokenCount,
	}

	// 序列化
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	// 设置Redis缓存
	err = s.redisClient.Set(common.BuildCacheKey(cacheKey), string(data), int(s.cacheTTL))
	if err != nil {
		return err
	}

	return nil
}

// Delete 删除缓存
func (s *StringCache) Delete(cacheKey string) error {
	if !s.enable {
		return nil
	}

	if s.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	return s.redisClient.Del(common.BuildCacheKey(cacheKey))
}

// Clear 清空所有缓存（慎用）
func (s *StringCache) Clear() error {
	if !s.enable {
		return nil
	}

	// 这里需要实现通配符删除，通常不建议在生产环境使用
	return fmt.Errorf("clear all cache is not supported for safety reasons")
}

// IsEnabled 检查是否启用
func (s *StringCache) IsEnabled() bool {
	return s.enable
}

// IsStreamCacheEnabled 检查是否启用流式缓存
func (s *StringCache) IsStreamCacheEnabled() bool {
	return s.cacheStream
}

// GetCacheTTL 获取缓存TTL
func (s *StringCache) GetCacheTTL() int64 {
	return s.cacheTTL
}

// GetMaxCacheSize 获取最大缓存大小
func (s *StringCache) GetMaxCacheSize() int {
	return s.maxCacheSize
}
