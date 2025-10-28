#!/bin/bash

# 微信公众号爬虫启动脚本

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_DIR"

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查Go环境
check_go() {
    if ! command -v go &> /dev/null; then
        log_error "Go环境未安装，请先安装Go 1.22+"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go版本: $GO_VERSION"
}

# 检查MongoDB
check_mongodb() {
    log_info "检查MongoDB连接..."
    # 这里可以添加MongoDB连接检查逻辑
}

# 创建必要的目录
create_dirs() {
    log_info "创建必要的目录..."
    mkdir -p logs
    mkdir -p chrome_data
}

# 编译项目
build_project() {
    log_info "编译项目..."
    if go build -o wechat-crawler cmd/main.go; then
        log_info "编译成功！"
    else
        log_error "编译失败！"
        exit 1
    fi
}

# 启动服务
start_service() {
    log_info "启动微信公众号爬虫服务..."
    log_info "========================================"
    log_info "提示："
    log_info "1. 首次运行需要扫码登录微信公众号平台"
    log_info "2. 登录成功后Cookie会自动保存"
    log_info "3. 按 Ctrl+C 可以停止服务"
    log_info "========================================"
    echo ""
    
    ./wechat-crawler
}

# 主函数
main() {
    log_info "========================================"
    log_info "   微信公众号爬虫系统 启动脚本"
    log_info "========================================"
    echo ""
    
    check_go
    create_dirs
    # check_mongodb
    build_project
    echo ""
    start_service
}

# 执行主函数
main

