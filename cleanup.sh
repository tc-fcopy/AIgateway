#!/bin/bash

# 清理脚本 - 停止测试进程并清理临时文件

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

log_info "=== 开始清理测试资源 ==="

# 1. 停止测试后端
log_info "--- 步骤 1：停止测试后端 ---"

# 检查是否有正在运行的 Go 进程
GO_PROCS=$(ps aux | grep -E "(test_ai_backend|test.*backend)" | grep -v grep | awk '{print $2}')

if [ -n "$GO_PROCS" ]; then
    log_info "停止 AI 后端进程"
    for PID in $GO_PROCS; do
        log_warning "  Killing PID: $PID"
        kill -9 $PID 2>/dev/null
    done
    log_success "后端进程已停止"
else
    log_info "未找到 AI 后端进程"
fi

# 检查 PID 文件
if [ -f gpt4.pid ]; then
    PID=$(cat gpt4.pid 2>/dev/null)
    if [ -n "$PID" ]; then
        log_info "检查 PID $PID"
        kill -9 $PID 2>/dev/null
    fi
    rm -f gpt4.pid
fi

if [ -f claude3.pid ]; then
    PID=$(cat claude3.pid 2>/dev/null)
    if [ -n "$PID" ]; then
        log_info "检查 PID $PID"
        kill -9 $PID 2>/dev/null
    fi
    rm -f claude3.pid
fi

# 2. 检查网关是否在运行
log_info "--- 步骤 2：检查网关状态 ---"

GATEWAY_PID=$(ps aux | grep -E "(main|AIGateway)" | grep -v grep | awk '{print $2}')

if [ -n "$GATEWAY_PID" ]; then
    log_warning "网关正在运行 (PID: $GATEWAY_PID) - 请手动停止"
else
    log_info "网关未运行"
fi

# 3. 清理临时文件
log_info "--- 步骤 3：清理临时文件 ---"

TMP_FILES=("*.log" "*.pid" "*.out" "*.tmp")

for PATTERN in "${TMP_FILES[@]}"; do
    if compgen -G "$PATTERN" > /dev/null; then
        log_warning "清理 $PATTERN"
        rm -f $PATTERN
    fi
done

# 4. 检查端口是否还在监听
log_info "--- 步骤 4：检查监听端口 ---"

# 检查端口 8080 (网关)
if lsof -ti :8080 > /dev/null; then
    log_warning "端口 8080 仍在监听 - 可能有进程未完全停止"
else
    log_info "端口 8080 已释放"
fi

# 检查测试后端端口
for PORT in 9001 9002; do
    if lsof -ti :$PORT > /dev/null; then
        log_warning "端口 $PORT 仍在监听 - 可能有进程未完全停止"
    else
        log_info "端口 $PORT 已释放"
    fi
done

# 5. 检查是否有僵尸进程
log_info "--- 步骤 5：检查系统资源 ---"

ZOMBIES=$(ps -A -ostat,ppid,pid,cmd | grep -e '^[Zz]')

if [ -n "$ZOMBIES" ]; then
    log_warning "发现僵尸进程，请检查："
    log_warning "$ZOMBIES"
else
    log_success "无僵尸进程"
fi

log_info "--- 清理完成 ---"
