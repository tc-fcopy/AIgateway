package ratelimit

// TokenRateLimitScript Token限流Lua脚本
const TokenRateLimitScript = `
local key = KEYS[1]
local window = tonumber(ARGV[1])
local tokens = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])

local current = redis.call("GET", key)
if current == false then
    redis.call("SET", key, tokens, "EX", window)
    return 1
end

current = tonumber(current)
if current + tokens > limit then
    return 0
end

redis.call("INCRBY", key, tokens)
return 1
`

// ResponsePhaseScript 响应阶段更新Token计数
const ResponsePhaseScript = `
local key = KEYS[1]
local tokens = tonumber(ARGV[1])
redis.call("INCRBY", key, tokens)
return 1
`

// ConsumeQuotaScript 原子扣减配额Lua脚本
const ConsumeQuotaScript = `
local key = KEYS[1]
local tokens = tonumber(ARGV[1])
local current = redis.call("GET", key)
if current == false then
    return 0
end
current = tonumber(current)
if current < tokens then
    return 0
end
redis.call("DECRBY", key, tokens)
return 1
`
