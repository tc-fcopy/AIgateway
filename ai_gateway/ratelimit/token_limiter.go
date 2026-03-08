package ratelimit

import (
	"errors"
	"fmt"
	"strconv"

	"gateway/ai_gateway/common"
	aigwctx "gateway/ai_gateway/context"
	"gateway/ai_gateway/token"
	"github.com/gin-gonic/gin"
)

// TokenLimiter performs distributed token-based limiting.
type TokenLimiter struct {
	redisClient *common.RedisClient
	enable      bool
}

func NewTokenLimiter(redisClient *common.RedisClient, enable bool) *TokenLimiter {
	return &TokenLimiter{redisClient: redisClient, enable: enable}
}

func (l *TokenLimiter) CheckLimit(c *gin.Context, serviceName, consumerName string, estimatedTokens int64) (bool, error) {
	if !l.enable || l.redisClient == nil {
		return true, nil
	}

	window, limit, ok := l.getWindowAndLimit(c)
	if !ok {
		return true, nil
	}

	redisKey := common.BuildTokenLimitKey(serviceName, consumerName, strconv.FormatInt(window, 10))
	result, err := l.redisClient.Eval(
		TokenRateLimitScript,
		1,
		[]string{redisKey},
		[]interface{}{window, estimatedTokens, limit},
	)
	if err != nil {
		return false, fmt.Errorf("token rate limit check failed: %w", err)
	}

	if intResult, ok := result.(int64); ok {
		return intResult == 1, nil
	}
	return false, errors.New("invalid rate limit response")
}

func (l *TokenLimiter) UpdateCount(c *gin.Context, serviceName, consumerName string, actualTokens int64) error {
	if !l.enable || l.redisClient == nil {
		return nil
	}

	window, _, ok := l.getWindowAndLimit(c)
	if !ok {
		return nil
	}

	redisKey := common.BuildTokenLimitKey(serviceName, consumerName, strconv.FormatInt(window, 10))
	_, err := l.redisClient.Eval(ResponsePhaseScript, 1, []string{redisKey}, []interface{}{actualTokens})
	if err != nil {
		return fmt.Errorf("token rate limit update failed: %w", err)
	}
	return nil
}

func (l *TokenLimiter) getWindowAndLimit(_ *gin.Context) (int64, int64, bool) {
	return 60, 10000, true
}

func (l *TokenLimiter) CheckMultipleWindows(c *gin.Context, serviceName, consumerName string, estimatedTokens int64) (bool, error) {
	if !l.enable || l.redisClient == nil {
		return true, nil
	}

	windows := l.getTimeWindows()
	limit, ok := l.getLimit(c)
	if !ok {
		return true, nil
	}

	for _, window := range windows {
		redisKey := common.BuildTokenLimitKey(serviceName, consumerName, strconv.FormatInt(window, 10))
		result, err := l.redisClient.Eval(
			TokenRateLimitScript,
			1,
			[]string{redisKey},
			[]interface{}{window, estimatedTokens, limit},
		)
		if err != nil {
			return false, fmt.Errorf("token rate limit check failed for window %d: %w", window, err)
		}

		if intResult, ok := result.(int64); ok && intResult == 0 {
			return false, nil
		}
	}

	return true, nil
}

func (l *TokenLimiter) getTimeWindows() []int64 {
	return []int64{60, 3600, 86400}
}

func (l *TokenLimiter) getLimit(_ *gin.Context) (int64, bool) {
	return 10000, true
}

func (l *TokenLimiter) RecordTokenUsage(c *gin.Context, usage *token.TokenUsage) error {
	c.Set(aigwctx.TokenUsageKey, usage)
	return nil
}

func (l *TokenLimiter) GetTokenUsageFromContext(c *gin.Context) (*token.TokenUsage, bool) {
	usage, exists := c.Get(aigwctx.TokenUsageKey)
	if !exists {
		return nil, false
	}
	out, ok := usage.(*token.TokenUsage)
	return out, ok
}

var (
	ErrRateLimitExceeded = errors.New("token rate limit exceeded")
	ErrServiceNotFound   = errors.New("service not found")
)
