# Dify Operator 使用指南

## 概述

Dify Operator 是基于 Kubernetes Operator 模式的 Dify 集群管理工具，通过自定义资源 `DifyCluster` 实现声明式配置和自动化管理。

## 架构对比

### 传统 Helm 模式 vs Operator 模式

| 特性 | 传统 Helm 模式 | Operator 模式 |
|------|---------------|---------------|
| **配置复杂度** | 3400+ 行 values.yaml | 100 行 DifyCluster |
| **自动化程度** | 手动管理 | 自动化管理 |
| **扩缩容** | 手动 HPA 配置 | 智能扩缩容 |
| **故障恢复** | 手动干预 | 自动自愈 |
| **版本升级** | 逐个组件升级 | 一键版本升级 |
| **学习成本** | 低 | 中等 |
| **运维效率** | 中等 | 高 |

## 快速开始

### 前置条件

- Kubernetes 1.18+
- Helm 3.x
- kubectl 配置正确

### 安装 Dify Operator

```bash
# 1. 添加 Helm 仓库
helm repo add dify-operator https://langgenius.github.io/dify-operator
helm repo update

# 2. 安装 Operator
helm install dify-operator dify-operator/dify-operator \
  --namespace dify-operator-system \
  --create-namespace

# 3. 验证安装
kubectl get pods -n dify-operator-system
```

### 部署 Dify 集群

```bash
# 1. 创建命名空间
kubectl create namespace dify-system

# 2. 创建 DifyCluster
cat << EOF | kubectl apply -f -
apiVersion: dify.ai/v1alpha1
kind: DifyCluster
metadata:
  name: dify-prod
  namespace: dify-system
spec:
  difyVersion: "1.7.1"
  components:
    api:
      enabled: true
      replicas: 2
      autoscaling:
        enabled: true
        minReplicas: 1
        maxReplicas: 10
        targetCPUUtilization: 70
    worker:
      enabled: true
      replicas: 3
    web:
      enabled: true
      replicas: 2
    sandbox:
      enabled: true
      replicas: 1
  storage:
    type: "local"
  database:
    postgresql:
      enabled: true
      database: "dify"
      username: "postgres"
      password: "your-secure-password"
    redis:
      enabled: true
      password: "your-secure-password"
EOF

# 3. 查看状态
kubectl get difycruster -n dify-system
kubectl get pods -n dify-system
```

## 配置迁移

### 从传统 Helm 迁移

如果您已有传统的 Dify Helm 部署，可以使用迁移工具：

```bash
# 1. 克隆项目
git clone https://github.com/langgenius/dify-operator.git
cd dify-operator

# 2. 运行迁移工具
python3 scripts/migrate-config.py \
  --input /path/to/old/values.yaml \
  --output difycruster.yaml

# 3. 应用新配置
kubectl apply -f difycruster.yaml
```

### 迁移步骤详解

1. **备份现有配置**
   ```bash
   helm get values your-dify-release > backup-values.yaml
   ```

2. **停止旧版本服务**（可选，支持蓝绿部署）
   ```bash
   helm uninstall your-dify-release
   ```

3. **应用新配置**
   ```bash
   kubectl apply -f difycruster.yaml
   ```

4. **验证迁移结果**
   ```bash
   kubectl get difycruster -o wide
   kubectl get pods -l app.kubernetes.io/name=dify
   ```

## 高级配置

### 自动扩缩容配置

```yaml
spec:
  components:
    api:
      autoscaling:
        enabled: true
        minReplicas: 2
        maxReplicas: 20
        targetCPUUtilization: 70
        targetMemoryUtilization: 80
    worker:
      autoscaling:
        enabled: true
        minReplicas: 3
        maxReplicas: 50
        targetCPUUtilization: 60
```

### 存储配置

#### S3 存储
```yaml
spec:
  storage:
    type: "s3"
    s3:
      endpoint: "https://s3.amazonaws.com"
      bucket: "dify-storage"
      accessKey: "your-access-key"
      secretKey: "your-secret-key"
      region: "us-east-1"
```

#### GCS 存储
```yaml
spec:
  storage:
    type: "gcs"
    gcs:
      bucket: "dify-storage"
      serviceAccountJsonBase64: "base64-encoded-json"
```

### 外部数据库配置

```yaml
spec:
  database:
    postgresql:
      enabled: true
      host: "your-postgres-host"
      database: "dify"
      username: "dify"
      password: "your-password"
    redis:
      enabled: true
      host: "your-redis-host"
      password: "your-password"
```

### 网络配置

```yaml
spec:
  networking:
    service:
      type: "LoadBalancer"
    ingress:
      enabled: true
      host: "dify.example.com"
      tls: true
```

## 运维操作

### 扩缩容操作

```bash
# 手动扩容 API 组件
kubectl patch difycruster dify-prod -n dify-system --type='merge' \
  -p='{"spec":{"components":{"api":{"replicas":5}}}}'

# 启用自动扩缩容
kubectl patch difycruster dify-prod -n dify-system --type='merge' \
  -p='{"spec":{"components":{"api":{"autoscaling":{"enabled":true}}}}}'
```

### 版本升级

```bash
# 升级到新版本
kubectl patch difycruster dify-prod -n dify-system --type='merge' \
  -p='{"spec":{"difyVersion":"1.8.0"}}'

# 查看升级状态
kubectl get difycruster dify-prod -n dify-system -w
```

### 监控和日志

```bash
# 查看集群状态
kubectl describe difycruster dify-prod -n dify-system

# 查看组件状态
kubectl get deployments -n dify-system
kubectl get pods -n dify-system

# 查看 Operator 日志
kubectl logs -n dify-operator-system deployment/dify-operator-controller-manager
```

### 故障排查

```bash
# 查看 DifyCluster 事件
kubectl describe difycruster dify-prod -n dify-system

# 查看 Pod 日志
kubectl logs -n dify-system deployment/dify-prod-api

# 查看 Operator 日志
kubectl logs -n dify-operator-system \
  deployment/dify-operator-controller-manager -f
```

## 性能优化

### 资源配置建议

#### 生产环境配置
```yaml
spec:
  components:
    api:
      replicas: 3
      resources:
        requests:
          cpu: "1000m"
          memory: "2Gi"
        limits:
          cpu: "4000m"
          memory: "8Gi"
      autoscaling:
        enabled: true
        minReplicas: 2
        maxReplicas: 20
    
    worker:
      replicas: 5
      resources:
        requests:
          cpu: "500m"
          memory: "1Gi"
        limits:
          cpu: "2000m"
          memory: "4Gi"
      autoscaling:
        enabled: true
        minReplicas: 3
        maxReplicas: 50
```

#### 开发环境配置
```yaml
spec:
  components:
    api:
      replicas: 1
      resources:
        requests:
          cpu: "100m"
          memory: "256Mi"
        limits:
          cpu: "500m"
          memory: "1Gi"
    
    worker:
      replicas: 1
      resources:
        requests:
          cpu: "100m"
          memory: "256Mi"
        limits:
          cpu: "500m"
          memory: "1Gi"
```

## 安全配置

### RBAC 配置

Operator 使用最小权限原则，仅具有管理 Dify 相关资源的权限：

```yaml
# Operator 权限范围
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["services", "configmaps"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["autoscaling"]
  resources: ["horizontalpodautoscalers"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

### 敏感信息管理

建议使用 Kubernetes Secret 管理敏感信息：

```bash
# 创建数据库密码 Secret
kubectl create secret generic dify-database \
  --from-literal=password=your-secure-password \
  -n dify-system

# 在 DifyCluster 中引用
spec:
  database:
    postgresql:
      passwordSecret:
        name: "dify-database"
        key: "password"
```

## 故障排查

### 常见问题

#### 1. Operator 无法启动
```bash
# 检查 RBAC 权限
kubectl auth can-i create deployments --as=system:serviceaccount:dify-operator-system:dify-operator

# 检查镜像拉取
kubectl describe pod -n dify-operator-system
```

#### 2. DifyCluster 创建失败
```bash
# 查看详细错误信息
kubectl describe difycruster your-cluster -n your-namespace

# 检查 Operator 日志
kubectl logs -n dify-operator-system deployment/dify-operator-controller-manager
```

#### 3. 组件无法启动
```bash
# 检查资源限制
kubectl describe pod your-pod -n your-namespace

# 检查依赖服务
kubectl get services -n your-namespace
```

## 卸载

### 完整卸载流程

```bash
# 1. 删除所有 DifyCluster
kubectl delete difycruster --all --all-namespaces

# 2. 卸载 Operator
helm uninstall dify-operator -n dify-operator-system

# 3. 删除 CRD（可选）
kubectl delete crd difyclusters.dify.ai

# 4. 删除命名空间
kubectl delete namespace dify-operator-system
```

## 参考资料

- [Kubernetes Operator 最佳实践](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [Helm Chart 开发指南](https://helm.sh/docs/chart_best_practices/)
- [Dify 官方文档](https://docs.dify.ai/)

## 支持和贡献

- **GitHub**: https://github.com/langgenius/dify-operator
- **Issues**: https://github.com/langgenius/dify-operator/issues
- **Discussions**: https://github.com/langgenius/dify-operator/discussions

欢迎提交 Issue 和 Pull Request 来改进项目！