#!/bin/bash

# 测试脚本 - 用于验证 AI 网关的模型转发功能
# 使用方法：bash run_test.sh

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# 启动时间
TEST_START=$(date +%s)

log_info "=== 开始 AI 网关模型转发功能测试 ==="
log_info "测试时间：$(date)"

# 1. 启动测试后端
log_info "--- 步骤 1：启动测试后端 ---"

# 检查是否已安装 Go
if ! command -v go &> /dev/null; then
    log_error "Go 未安装，请先安装 Go"
    exit 1
fi

# 启动两个不同的 AI 模型后端
log_info "启动 GPT-4 模拟后端（端口 9001）..."
go run test_ai_backend.go -port 9001 -model "gpt-4" > gpt4_backend.log 2>&1 &
GPT4_PID=$!
echo $GPT4_PID > gpt4.pid
log_success "GPT-4 后端 PID: $GPT4_PID"

log_info "启动 Claude-3 模拟后端（端口 9002）..."
go run test_ai_backend.go -port 9002 -model "claude-3" > claude3_backend.log 2>&1 &
CLAUDE3_PID=$!
echo $CLAUDE3_PID > claude3.pid
log_success "Claude-3 后端 PID: $CLAUDE3_PID"

# 等待后端启动
log_info "等待后端启动..."
sleep 2

# 检查后端是否在运行
log_info "--- 步骤 2：检查后端是否在线 ---"

for i in 1 2 3 4 5; do
    GPT4_RESP=$(curl -s "http://localhost:9001/")
    CLAUDE3_RESP=$(curl -s "http://localhost:9002/")

    if [ -n "$GPT4_RESP" ] && [ -n "$CLAUDE3_RESP" ]; then
        log_success "所有后端均在线！"
        log_info "  GPT-4: $GPT4_RESP"
        log_info "  Claude-3: $CLAUDE3_RESP"
        break
    fi

    log_warning "后端启动中，第 $i 次尝试..."
    sleep 1
done

if [ -z "$GPT4_RESP" ]; then
    log_error "GPT-4 后端未响应，可能启动失败"
fi

if [ -z "$CLAUDE3_RESP" ]; then
    log_error "Claude-3 后端未响应，可能启动失败"
fi

# 3. 启动网关
log_info "--- 步骤 3：启动网关 ---"
log_warning "⚠️  请确保你的网关配置正确，并且连接到了这些测试后端"

# 4. 运行测试
log_info "--- 步骤 4：运行测试 ---"

TEST_DIR=$(dirname "$0")

# 测试 1：测试网关连接
log_info "测试网关是否可连接："

GATEWAY_RESP=$(curl -s "http://localhost:8080/ping")
if [ -n "$GATEWAY_RESP" ]; then
    log_success "网关可连接：$GATEWAY_RESP"
else
    log_error "网关未响应，可能未启动"
    log_warning "请检查："
    log_warning "1. 网关是否已启动"
    log_warning "2. 端口 8080 是否被占用"
fi

# 测试 2：测试模型路由 - 发送 GPT 请求
log_info "测试模型路由：发送 GPT 请求"

GPT_TEST=$(curl -s "http://localhost:8080/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello, what model are you?"}]
  }')

if [ -n "$GPT_TEST" ]; then
    log_success "请求成功！"
    log_info "响应内容：$(echo "$GPT_TEST" | head -5)"

    # 检查是否有 model 字段
    FOUND_MODEL=$(echo "$GPT_TEST" | grep -o '"model":"[^"]*"' | cut -d'"' -f4)
    if [ -n "$FOUND_MODEL" ]; then
        if [ "$FOUND_MODEL" = "gpt-4" ]; then
            log_success "✅ 正确路由到 GPT-4：响应模型 = $FOUND_MODEL"
        else
            log_error "❌ 路由到错误的模型：响应模型 = $FOUND_MODEL"
            log_warning "期望：gpt-4"
        fi
    fi
else
    log_error "请求失败"
fi

# 测试 3：测试模型路由 - 发送 Claude 请求
log_info "测试模型路由：发送 Claude 请求"

CLAUDE_TEST=$(curl -s "http://localhost:8080/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-2",
    "messages": [{"role": "user", "content": "Hello, what model are you?"}]
  }')

if [ -n "$CLAUDE_TEST" ]; then
    log_success "请求成功！"
    log_info "响应内容：$(echo "$CLAUDE_TEST" | head -5)"

    FOUND_MODEL=$(echo "$CLAUDE_TEST" | grep -o '"model":"[^"]*"' | cut -d'"' -f4)
    if [ -n "$FOUND_MODEL" ]; then
        if [ "$FOUND_MODEL" = "claude-3" ]; then
            log_success "✅ 正确路由到 Claude-3：响应模型 = $FOUND_MODEL"
        else
            log_error "❌ 路由到错误的模型：响应模型 = $FOUND_MODEL"
            log_warning "期望：claude-3"
        fi
    fi
else
    log_error "请求失败"
fi

# 5. 检查日志
log_info "--- 步骤 5：检查测试结果 ---"

log_warning "📊 后端日志已保存到："
log_warning "  gpt4_backend.log: GPT-4 后端详细日志"
log_warning "  claude3_backend.log: Claude-3 后端详细日志"

# 显示一些关键日志条目
log_info "--- 后端日志 ---"
if [ -f gpt4_backend.log ]; then
    log_info "GPT-4 后端日志（最后 3 条）："
    tail -3 gpt4_backend.log 2>/dev/null || log_warning "无法读取 GPT-4 后端日志"
fi
if [ -f claude3_backend.log ]; then
    log_info "Claude-3 后端日志（最后 3 条）："
    tail -3 claude3_backend.log 2>/dev/null || log_warning "无法读取 Claude-3 后端日志"
fi

# 计算测试时长
TEST_END=$(date +%s)
TEST_DURATION=$((TEST_END - TEST_START))

# 清理临时文件
rm -f gpt4.pid claude3.pid

log_info "--- 测试完成 ---"
log_success "测试总耗时：${TEST_DURATION} 秒"

log_warning "⚠️  记得："
log_warning "1. 使用 Ctrl+C 停止网关"
log_warning "2. 停止后端服务器"
log_warning "3. 运行清理脚本：./cleanup.sh"
