#!/bin/bash

# TemJob 启动脚本
# 用于快速启动 TemJob 服务及其依赖

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        print_error "$1 命令未找到，请先安装 $1"
        exit 1
    fi
}

# 检查端口是否被占用
check_port() {
    if lsof -Pi :$1 -sTCP:LISTEN -t >/dev/null 2>&1; then
        print_warning "端口 $1 已被占用"
        return 1
    fi
    return 0
}

# 等待服务启动
wait_for_service() {
    local host=$1
    local port=$2
    local service_name=$3
    local max_attempts=30
    local attempt=1

    print_info "等待 $service_name 启动..."
    
    while [ $attempt -le $max_attempts ]; do
        if nc -z $host $port 2>/dev/null; then
            print_success "$service_name 已启动"
            return 0
        fi
        
        printf "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo
    print_error "$service_name 启动超时"
    return 1
}

# 启动模式
MODE=${1:-"development"}

print_info "启动 TemJob ($MODE 模式)..."

# 检查必要的命令
print_info "检查系统依赖..."
check_command "docker"
check_command "docker-compose"
check_command "curl"

# 根据模式执行不同的启动逻辑
case $MODE in
    "development"|"dev")
        print_info "启动开发环境..."
        
        # 检查Go环境
        check_command "go"
        
        # 检查配置文件
        if [ ! -f "config.yaml" ]; then
            print_warning "config.yaml 不存在，复制默认配置..."
            cp config.yaml.example config.yaml 2>/dev/null || {
                print_error "默认配置文件不存在，请手动创建 config.yaml"
                exit 1
            }
        fi
        
        # 启动依赖服务 (Redis + MySQL)
        print_info "启动依赖服务..."
        docker-compose up -d redis mysql
        
        # 等待服务启动
        wait_for_service "localhost" "6379" "Redis"
        wait_for_service "localhost" "3306" "MySQL"
        
        # 安装Go依赖
        print_info "安装Go依赖..."
        go mod tidy
        
        # 检查端口
        if ! check_port 8088; then
            print_error "TemJob 端口 8088 被占用，请先停止占用该端口的服务"
            exit 1
        fi
        
        # 启动TemJob
        print_info "启动 TemJob 服务..."
        go run main.go --config config.yaml &
        TEMJOB_PID=$!
        
        # 等待TemJob启动
        sleep 5
        if wait_for_service "localhost" "8088" "TemJob"; then
            print_success "开发环境启动成功！"
            print_info "Web UI: http://localhost:8088"
            print_info "API 文档: http://localhost:8088/api/v1"
            print_info "按 Ctrl+C 停止服务"
            
            # 等待用户停止
            trap "print_info '正在停止服务...'; kill $TEMJOB_PID 2>/dev/null; docker-compose stop redis mysql; exit 0" INT
            wait $TEMJOB_PID
        else
            print_error "TemJob 启动失败"
            exit 1
        fi
        ;;
        
    "production"|"prod")
        print_info "启动生产环境..."
        
        # 检查环境变量文件
        if [ ! -f ".env" ]; then
            print_warning ".env 文件不存在，复制示例文件..."
            cp .env.example .env
            print_warning "请编辑 .env 文件配置生产环境参数"
        fi
        
        # 构建并启动所有服务
        print_info "构建并启动服务..."
        docker-compose --env-file .env up -d --build
        
        # 等待服务启动
        wait_for_service "localhost" "8088" "TemJob"
        
        print_success "生产环境启动成功！"
        print_info "Web UI: http://localhost:8088"
        print_info "Redis 管理: http://localhost:8081 (admin/admin123)"
        print_info "MySQL 管理: http://localhost:8080"
        print_info "监控面板: http://localhost:3000 (admin/admin123)"
        ;;
        
    "docker")
        print_info "启动 Docker 环境..."
        
        # 仅启动必要服务
        docker-compose up -d temjob redis mysql
        
        wait_for_service "localhost" "8088" "TemJob"
        
        print_success "Docker 环境启动成功！"
        print_info "Web UI: http://localhost:8088"
        ;;
        
    "cluster")
        print_info "启动集群环境..."
        
        # 启动多个TemJob实例
        docker-compose up -d --scale temjob=3 --scale temjob-worker=5
        
        wait_for_service "localhost" "8088" "TemJob Cluster"
        
        print_success "集群环境启动成功！"
        print_info "Load Balancer: http://localhost"
        print_info "直接访问: http://localhost:8088"
        ;;
        
    "stop")
        print_info "停止所有服务..."
        docker-compose down
        print_success "所有服务已停止"
        exit 0
        ;;
        
    "restart")
        print_info "重启所有服务..."
        docker-compose restart
        wait_for_service "localhost" "8088" "TemJob"
        print_success "所有服务已重启"
        ;;
        
    "logs")
        print_info "显示服务日志..."
        docker-compose logs -f temjob
        ;;
        
    "status")
        print_info "检查服务状态..."
        docker-compose ps
        
        # 检查服务健康状态
        if curl -s http://localhost:8088/api/v1/stats >/dev/null 2>&1; then
            print_success "TemJob 服务运行正常"
        else
            print_error "TemJob 服务无响应"
        fi
        ;;
        
    "clean")
        print_info "清理所有数据..."
        read -p "这将删除所有数据，确定继续吗？(y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            docker-compose down -v
            docker system prune -f
            print_success "清理完成"
        else
            print_info "取消清理操作"
        fi
        ;;
        
    "help"|"--help"|"-h")
        echo "TemJob 启动脚本"
        echo ""
        echo "用法: $0 [模式]"
        echo ""
        echo "可用模式:"
        echo "  development, dev  - 开发模式 (默认)"
        echo "  production, prod  - 生产模式"
        echo "  docker           - Docker 模式"
        echo "  cluster          - 集群模式"
        echo "  stop             - 停止所有服务"
        echo "  restart          - 重启所有服务"
        echo "  logs             - 查看日志"
        echo "  status           - 检查状态"
        echo "  clean            - 清理数据"
        echo "  help             - 显示帮助"
        echo ""
        echo "示例:"
        echo "  $0 dev           # 启动开发环境"
        echo "  $0 prod          # 启动生产环境"
        echo "  $0 stop          # 停止服务"
        echo "  $0 status        # 检查状态"
        ;;
        
    *)
        print_error "未知模式: $MODE"
        print_info "使用 '$0 help' 查看可用模式"
        exit 1
        ;;
esac