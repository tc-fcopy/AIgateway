package public

import (
	"fmt"
	"golang.org/x/time/rate"
	"sync"
	"sync/atomic"
	"time"
)

var FlowLimiterHandler *FlowLimiter

type FlowLimiter struct {
	FlowLmiterMap    map[string]*FlowLimiterItem
	FlowLmiterSlice  []*FlowLimiterItem
	ShardedBucketMap map[string]*ShardedTokenBucket // 新增：分片令牌桶映射
	Locker           sync.RWMutex
}

type FlowLimiterItem struct {
	ServiceName string
	Limter      *rate.Limiter
}

// ==================== 分片令牌桶 ====================
type ShardedTokenBucket struct {
	shards     []*TokenBucketShard
	shardCount int
	serviceQPS int64
}

type TokenBucketShard struct {
	remainingTokens int64
	lastRefillTime  int64
	mutex           sync.Mutex
}

func NewFlowLimiter() *FlowLimiter {
	return &FlowLimiter{
		FlowLmiterMap:    map[string]*FlowLimiterItem{},
		FlowLmiterSlice:  []*FlowLimiterItem{},
		ShardedBucketMap: map[string]*ShardedTokenBucket{},
		Locker:           sync.RWMutex{},
	}
}

func init() {
	FlowLimiterHandler = NewFlowLimiter()
}

func (counter *FlowLimiter) GetLimiter(serverName string, qps float64) (*rate.Limiter, error) {
	// fmt.Println(len(counter.FlowLmiterSlice))

	// 1. 读操作加读锁（同时查Map，比Slice遍历高效）
	counter.Locker.RLock()
	item, ok := counter.FlowLmiterMap[serverName]
	counter.Locker.RUnlock()
	if ok {
		return item.Limter, nil
	}

	// 2. 写操作加写锁 + 双重检查（避免重复创建）
	counter.Locker.Lock()
	defer counter.Locker.Unlock()
	// 双重检查：防止加锁期间已有其他goroutine创建了限流器
	item, ok = counter.FlowLmiterMap[serverName]
	if ok {
		return item.Limter, nil
	}

	// 3. 创建限流器（burst=300，即1倍，或按你的需求设3倍，但先保证单实例）
	burst := int(qps) // 先设为1倍，验证限流生效；若要3倍则改为int(qps*3)
	newLimiter := rate.NewLimiter(rate.Limit(qps), burst)
	item = &FlowLimiterItem{
		ServiceName: serverName,
		Limter:      newLimiter,
	}
	// 4. 同时更新Map和Slice（加锁后操作）
	counter.FlowLmiterMap[serverName] = item
	counter.FlowLmiterSlice = append(counter.FlowLmiterSlice, item)
	return newLimiter, nil
}

// =================================================
// 分片令牌桶

// GetShardedLimiter 获取分片令牌桶限流器
func (counter *FlowLimiter) GetShardedLimiter(serverName string, qps float64) (*ShardedTokenBucket, error) {
	counter.Locker.RLock()
	bucket, ok := counter.ShardedBucketMap[serverName]
	counter.Locker.RUnlock()

	if ok {
		return bucket, nil
	}

	counter.Locker.Lock()
	defer counter.Locker.Unlock()

	// 双重检查
	bucket, ok = counter.ShardedBucketMap[serverName]
	if ok {
		return bucket, nil
	}

	// 创建分片令牌桶
	bucket = counter.createShardedTokenBucket(int64(qps))
	counter.ShardedBucketMap[serverName] = bucket
	fmt.Printf("创建分片令牌桶限流器，QPS: %v\n", qps)
	return bucket, nil
}

// createShardedTokenBucket 创建分片令牌桶
func (counter *FlowLimiter) createShardedTokenBucket(serviceQPS int64) *ShardedTokenBucket {
	shardCount := 4
	shards := make([]*TokenBucketShard, shardCount)

	singleShardRate := serviceQPS / int64(shardCount)
	if singleShardRate <= 0 {
		singleShardRate = 1
	}

	for i := 0; i < shardCount; i++ {
		shards[i] = &TokenBucketShard{
			remainingTokens: singleShardRate * 2, // 增大初始令牌数
			lastRefillTime:  time.Now().UnixNano(),
		}
	}

	return &ShardedTokenBucket{
		shards:     shards,
		shardCount: shardCount,
		serviceQPS: serviceQPS,
	}
}

// AllowSharded 分片令牌桶的限流判断
func (counter *FlowLimiter) AllowSharded(bucket *ShardedTokenBucket) bool {
	// 使用随机数选择分片（避免热点）
	shardIndex := int(time.Now().UnixNano()) % bucket.shardCount
	shard := bucket.shards[shardIndex]

	now := time.Now().UnixNano()
	shard.mutex.Lock()

	singleShardRate := bucket.serviceQPS / int64(bucket.shardCount)
	if singleShardRate <= 0 {
		singleShardRate = 1
	}

	elapsed := now - shard.lastRefillTime
	seconds := float64(elapsed) / 1e9
	tokensToAdd := int64(seconds * float64(singleShardRate))

	// 确保至少补充1个token（如果时间间隔大于0）
	if elapsed > 0 && tokensToAdd == 0 && singleShardRate > 0 {
		tokensToAdd = 1
	}

	if tokensToAdd > 0 {
		newTokens := atomic.AddInt64(&shard.remainingTokens, tokensToAdd)
		maxTokens := singleShardRate * 4 // 增大桶容量
		if newTokens > maxTokens {
			atomic.StoreInt64(&shard.remainingTokens, maxTokens)
		}
		shard.lastRefillTime = now
	}

	remaining := atomic.LoadInt64(&shard.remainingTokens)
	if remaining > 0 {
		newRemaining := atomic.AddInt64(&shard.remainingTokens, -1)
		if newRemaining >= 0 {
			shard.mutex.Unlock()
			return true
		} else {
			// 消耗失败，回滚
			atomic.AddInt64(&shard.remainingTokens, 1)
			shard.mutex.Unlock()
			return false
		}
	} else {
		shard.mutex.Unlock()
		return false
	}
}
