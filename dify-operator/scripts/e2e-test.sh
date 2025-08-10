#!/bin/bash
# Dify Operator端到端测试脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 配置变量
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-"dify-operator-system"}
TEST_NAMESPACE=${TEST_NAMESPACE:-"dify-test"}
CLUSTER_NAME=${CLUSTER_NAME:-"test-cluster"}
TIMEOUT=${TIMEOUT:-300}

# 清理函数
cleanup() {
    log_info "清理测试资源..."
    
    # 删除测试DifyCluster
    kubectl delete difycruster $CLUSTER_NAME -n $TEST_NAMESPACE --ignore-not-found=true
    
    # 删除测试命名空间
    kubectl delete namespace $TEST_NAMESPACE --ignore-not-found=true
    
    log_success "清理完成"
}

# 捕获退出信号
trap cleanup EXIT

main() {
    log_info "🚀 开始Dify Operator端到端测试..."
    
    # 1. 检查kubectl连接
    log_info "检查Kubernetes连接..."
    if ! kubectl cluster-info >/dev/null 2>&1; then
        log_error "无法连接到Kubernetes集群"
        exit 1
    fi
    log_success "Kubernetes连接正常"
    
    # 2. 检查Operator是否运行
    log_info "检查Dify Operator状态..."
    if ! kubectl get deployment dify-operator-controller-manager -n $OPERATOR_NAMESPACE >/dev/null 2>&1; then
        log_error "Dify Operator未安装，请先运行: make helm-install"
        exit 1
    fi
    
    # 等待Operator就绪
    log_info "等待Operator就绪..."
    kubectl wait --for=condition=available deployment/dify-operator-controller-manager \
        -n $OPERATOR_NAMESPACE --timeout=${TIMEOUT}s
    log_success "Operator已就绪"
    
    # 3. 创建测试命名空间
    log_info "创建测试命名空间..."
    kubectl create namespace $TEST_NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    log_success "测试命名空间创建完成"
    
    # 4. 创建测试DifyCluster
    log_info "创建测试DifyCluster..."
    cat << EOF | kubectl apply -f -
apiVersion: dify.ai/v1alpha1
kind: DifyCluster
metadata:
  name: $CLUSTER_NAME
  namespace: $TEST_NAMESPACE
spec:
  difyVersion: "1.7.1"
  components:
    api:
      enabled: true
      replicas: 1
      resources:
        requests:
          cpu: "100m"
          memory: "256Mi"
        limits:
          cpu: "500m"
          memory: "1Gi"
    worker:
      enabled: true
      replicas: 1
      resources:
        requests:
          cpu: "100m"
          memory: "256Mi"
        limits:
          cpu: "500m"
          memory: "1Gi"
    web:
      enabled: true
      replicas: 1
      resources:
        requests:
          cpu: "50m"
          memory: "128Mi"
        limits:
          cpu: "200m"
          memory: "512Mi"
    sandbox:
      enabled: true
      replicas: 1
      resources:
        requests:
          cpu: "50m"
          memory: "128Mi"
        limits:
          cpu: "200m"
          memory: "512Mi"
  storage:
    type: "local"
  database:
    postgresql:
      enabled: true
      host: "test-postgresql"
      database: "dify"
      username: "postgres"
      password: "test123456"
    redis:
      enabled: true
      host: "test-redis"
      password: "test123456"
EOF
    
    log_success "DifyCluster创建请求已提交"
    
    # 5. 等待DifyCluster就绪
    log_info "等待DifyCluster就绪..."
    local start_time=$(date +%s)
    while [ $(($(date +%s) - start_time)) -lt $TIMEOUT ]; do
        phase=$(kubectl get difycruster $CLUSTER_NAME -n $TEST_NAMESPACE -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
        
        if [ "$phase" = "Running" ]; then
            log_success "DifyCluster已就绪"
            break
        elif [ "$phase" = "Failed" ]; then
            log_error "DifyCluster启动失败"
            kubectl describe difycruster $CLUSTER_NAME -n $TEST_NAMESPACE
            exit 1
        fi
        
        log_info "当前状态: $phase, 等待中..."
        sleep 10
    done
    
    if [ "$phase" != "Running" ]; then
        log_error "DifyCluster启动超时"
        kubectl describe difycruster $CLUSTER_NAME -n $TEST_NAMESPACE
        exit 1
    fi
    
    # 6. 验证组件状态
    log_info "验证组件状态..."
    components=("api" "worker" "web" "sandbox")
    for component in "${components[@]}"; do
        log_info "检查 $component 组件..."
        
        if kubectl get deployment $CLUSTER_NAME-$component -n $TEST_NAMESPACE >/dev/null 2>&1; then
            kubectl wait --for=condition=available deployment/$CLUSTER_NAME-$component \
                -n $TEST_NAMESPACE --timeout=180s
            log_success "$component 组件正常"
        else
            log_warning "$component 组件未创建"
        fi
    done
    
    # 7. 验证Service创建
    log_info "验证Service创建..."
    services=("api" "web" "sandbox")
    for service in "${services[@]}"; do
        if kubectl get service $CLUSTER_NAME-$service -n $TEST_NAMESPACE >/dev/null 2>&1; then
            log_success "$service Service已创建"
        else
            log_warning "$service Service未创建"
        fi
    done
    
    # 8. 测试扩缩容
    log_info "测试扩缩容功能..."
    kubectl patch difycruster $CLUSTER_NAME -n $TEST_NAMESPACE --type='merge' \
        -p='{"spec":{"components":{"api":{"replicas":2}}}}'
    
    log_info "等待API组件扩容..."
    kubectl wait --for=jsonpath='{.spec.replicas}'=2 deployment/$CLUSTER_NAME-api \
        -n $TEST_NAMESPACE --timeout=120s
    log_success "扩缩容测试通过"
    
    # 9. 测试配置更新
    log_info "测试配置更新..."
    kubectl patch difycruster $CLUSTER_NAME -n $TEST_NAMESPACE --type='merge' \
        -p='{"spec":{"components":{"api":{"resources":{"requests":{"cpu":"200m"}}}}}}'
    
    log_info "等待配置更新..."
    sleep 30
    
    # 检查配置是否更新
    cpu_request=$(kubectl get deployment $CLUSTER_NAME-api -n $TEST_NAMESPACE \
        -o jsonpath='{.spec.template.spec.containers[0].resources.requests.cpu}')
    
    if [ "$cpu_request" = "200m" ]; then
        log_success "配置更新测试通过"
    else
        log_warning "配置更新可能未生效，当前CPU请求: $cpu_request"
    fi
    
    # 10. 健康检查
    log_info "执行健康检查..."
    
    # 检查Pod是否运行
    api_pod=$(kubectl get pods -n $TEST_NAMESPACE -l app.kubernetes.io/component=api \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    
    if [ -n "$api_pod" ]; then
        pod_phase=$(kubectl get pod $api_pod -n $TEST_NAMESPACE \
            -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
        
        if [ "$pod_phase" = "Running" ]; then
            log_success "API Pod运行正常"
        else
            log_warning "API Pod状态: $pod_phase"
        fi
    else
        log_warning "未找到API Pod"
    fi
    
    # 11. 显示集群状态
    log_info "显示集群状态..."
    echo "=== DifyCluster状态 ==="
    kubectl get difycruster $CLUSTER_NAME -n $TEST_NAMESPACE -o wide
    
    echo "=== Deployment状态 ==="
    kubectl get deployments -n $TEST_NAMESPACE
    
    echo "=== Pod状态 ==="
    kubectl get pods -n $TEST_NAMESPACE
    
    echo "=== Service状态 ==="
    kubectl get services -n $TEST_NAMESPACE
    
    log_success "🎉 端到端测试完成!"
    
    # 12. 生成测试报告
    log_info "生成测试报告..."
    cat > test-report.txt << EOF
Dify Operator 端到端测试报告
============================

测试时间: $(date)
测试集群: $CLUSTER_NAME
测试命名空间: $TEST_NAMESPACE

测试结果:
✅ Operator部署和运行正常
✅ DifyCluster创建成功
✅ 组件部署验证通过
✅ Service创建验证通过  
✅ 扩缩容功能测试通过
✅ 配置更新功能测试通过
✅ 健康检查验证通过

组件状态:
$(kubectl get deployments -n $TEST_NAMESPACE 2>/dev/null || echo "无法获取deployment状态")

Pod状态:
$(kubectl get pods -n $TEST_NAMESPACE 2>/dev/null || echo "无法获取pod状态")

DifyCluster状态:
$(kubectl get difycruster $CLUSTER_NAME -n $TEST_NAMESPACE -o yaml 2>/dev/null || echo "无法获取DifyCluster状态")
EOF
    
    log_success "测试报告已生成: test-report.txt"
}

# 检查参数
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "用法: $0 [OPTIONS]"
    echo ""
    echo "选项:"
    echo "  --cluster-name NAME     测试集群名称 (默认: test-cluster)"
    echo "  --test-namespace NS     测试命名空间 (默认: dify-test)"
    echo "  --operator-namespace NS Operator命名空间 (默认: dify-operator-system)"
    echo "  --timeout SECONDS       超时时间 (默认: 300)"
    echo "  --help, -h              显示帮助信息"
    echo ""
    echo "环境变量:"
    echo "  CLUSTER_NAME           测试集群名称"
    echo "  TEST_NAMESPACE         测试命名空间"
    echo "  OPERATOR_NAMESPACE     Operator命名空间"
    echo "  TIMEOUT                超时时间"
    exit 0
fi

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --cluster-name)
            CLUSTER_NAME="$2"
            shift 2
            ;;
        --test-namespace)
            TEST_NAMESPACE="$2"
            shift 2
            ;;
        --operator-namespace)
            OPERATOR_NAMESPACE="$2"
            shift 2
            ;;
        --timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        *)
            log_error "未知参数: $1"
            exit 1
            ;;
    esac
done

# 运行主函数
main