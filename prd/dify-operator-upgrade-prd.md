# Dify-Helm Operator架构升级PRD v1.0.0

## 1. 项目背景

### 1.1 现状分析

当前dify-helm项目采用**传统Helm Chart + Kubernetes原生资源**模式，存在以下局限性：

**🔍 技术架构分析**：
- **直接资源管理**：通过Deployment、Service、ConfigMap等原生资源直接部署
- **缺乏抽象层**：每个组件(API、Worker、Web、Sandbox等)需要单独配置和维护
- **扩缩容复杂**：依赖HPA进行被动扩缩容，缺乏业务逻辑感知
- **生命周期管理**：组件间依赖关系需要手动维护，缺乏统一协调

**📊 当前架构痛点**：
1. **配置复杂度高**：8个核心组件需要独立配置，values.yaml超过3400行
2. **运维负担重**：组件间依赖需要手动协调，故障恢复依赖人工干预
3. **扩展性受限**：难以支持复杂的业务扩缩容策略和自定义调度逻辑
4. **版本管理困难**：组件版本升级需要逐一处理，容易出现兼容性问题

### 1.2 需求来源

**业务驱动**：
- **企业级需求**：大规模部署场景下需要更强的自动化管理能力
- **运维效率提升**：减少人工干预，提高系统可靠性和运维效率
- **技术演进需求**：向Cloud Native最佳实践靠拢

**技术驱动**：
- **KubeRay成功案例**：Ray生态系统通过Operator模式实现了优秀的集群管理
- **Operator模式成熟**：Kubernetes社区Operator模式已成为复杂应用管理的标准
- **扩展性需求**：未来需要支持更多自定义逻辑和业务场景

## 2. 产品目标

### 2.1 主要目标

1. **架构现代化**：将dify-helm从传统模式升级为CRD + Operator + Helm三层架构
2. **操作简化**：通过DifyCluster CRD实现声明式配置，简化部署和运维复杂度
3. **自动化增强**：实现智能扩缩容、故障自愈、版本管理等高级功能
4. **兼容性保证**：确保升级过程平滑，现有配置能够无缝迁移
5. **生态集成**：为未来集成更多AI/ML工具链奠定基础

### 2.2 成功指标

- **部署简化**：配置复杂度降低70%（从3400行配置减少到1000行以内）
- **运维效率**：故障恢复时间从15分钟缩短到2分钟以内
- **扩缩容性能**：响应时间从5分钟提升到30秒以内
- **版本升级**：支持一键版本升级，成功率>99%
- **资源利用率**：通过智能调度提升资源利用率20%

## 3. 功能需求

### 3.1 核心架构设计

#### 3.1.1 DifyCluster CRD设计

**功能描述**：定义Dify集群的声明式API，抽象化管理所有组件配置

**CRD Schema设计**：
```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: difyclusters.dify.ai
spec:
  group: dify.ai
  versions:
  - name: v1alpha1
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              # Dify版本控制
              difyVersion:
                type: string
                enum: ["1.7.1", "1.8.0"]
                default: "1.7.1"
              
              # 核心组件配置
              components:
                type: object
                properties:
                  api:
                    type: object
                    properties:
                      enabled:
                        type: boolean
                        default: true
                      replicas:
                        type: integer
                        minimum: 1
                        maximum: 100
                        default: 2
                      autoscaling:
                        type: object
                        properties:
                          enabled:
                            type: boolean
                            default: true
                          minReplicas:
                            type: integer
                            default: 1
                          maxReplicas:
                            type: integer
                            default: 20
                          targetCPUUtilization:
                            type: integer
                            default: 70
                      resources:
                        type: object
                        properties:
                          requests:
                            type: object
                            properties:
                              cpu:
                                type: string
                                default: "500m"
                              memory:
                                type: string
                                default: "1Gi"
                          limits:
                            type: object
                            properties:
                              cpu:
                                type: string
                                default: "2000m"
                              memory:
                                type: string
                                default: "4Gi"
                  
                  worker:
                    type: object
                    properties:
                      enabled:
                        type: boolean
                        default: true
                      replicas:
                        type: integer
                        default: 3
                      autoscaling:
                        type: object
                        properties:
                          enabled:
                            type: boolean
                            default: true
                          minReplicas:
                            type: integer
                            default: 2
                          maxReplicas:
                            type: integer
                            default: 50
                  
                  web:
                    type: object
                    properties:
                      enabled:
                        type: boolean
                        default: true
                      replicas:
                        type: integer
                        default: 2
                  
                  sandbox:
                    type: object
                    properties:
                      enabled:
                        type: boolean
                        default: true
                      replicas:
                        type: integer
                        default: 1
                  
                  pluginDaemon:
                    type: object
                    properties:
                      enabled:
                        type: boolean
                        default: true
                      replicas:
                        type: integer
                        default: 1
              
              # 存储配置
              storage:
                type: object
                properties:
                  type:
                    type: string
                    enum: ["local", "s3", "gcs", "azure"]
                    default: "local"
                  s3:
                    type: object
                    properties:
                      endpoint:
                        type: string
                      bucket:
                        type: string
                      accessKey:
                        type: string
                      secretKey:
                        type: string
              
              # 数据库配置
              database:
                type: object
                properties:
                  type:
                    type: string
                    enum: ["internal", "external"]
                    default: "internal"
                  postgresql:
                    type: object
                    properties:
                      enabled:
                        type: boolean
                        default: true
                      host:
                        type: string
                      database:
                        type: string
                        default: "dify"
                      username:
                        type: string
                      password:
                        type: string
                  redis:
                    type: object
                    properties:
                      enabled:
                        type: boolean
                        default: true
                      host:
                        type: string
                      password:
                        type: string
              
              # 网络配置
              networking:
                type: object
                properties:
                  ingress:
                    type: object
                    properties:
                      enabled:
                        type: boolean
                        default: false
                      host:
                        type: string
                      tls:
                        type: boolean
                        default: false
                  service:
                    type: object
                    properties:
                      type:
                        type: string
                        enum: ["ClusterIP", "NodePort", "LoadBalancer"]
                        default: "ClusterIP"
          
          status:
            type: object
            properties:
              phase:
                type: string
                enum: ["Pending", "Running", "Upgrading", "Failed"]
              conditions:
                type: array
                items:
                  type: object
                  properties:
                    type:
                      type: string
                    status:
                      type: string
                    reason:
                      type: string
                    message:
                      type: string
                    lastTransitionTime:
                      type: string
              componentStatus:
                type: object
                properties:
                  api:
                    type: object
                    properties:
                      ready:
                        type: boolean
                      replicas:
                        type: integer
                      readyReplicas:
                        type: integer
                  worker:
                    type: object
                    properties:
                      ready:
                        type: boolean
                      replicas:
                        type: integer
                      readyReplicas:
                        type: integer
              observedGeneration:
                type: integer
```

#### 3.1.2 Dify Operator核心功能

**功能描述**：监听DifyCluster资源变化，自动管理Dify组件的生命周期

**控制器逻辑设计**：
```go
// pkg/controllers/difycruster_controller.go
package controllers

import (
    "context"
    "time"
    
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    "k8s.io/apimachinery/pkg/runtime"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    
    difyv1alpha1 "github.com/langgenius/dify-operator/api/v1alpha1"
)

// DifyClusterReconciler reconciles a DifyCluster object
type DifyClusterReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=dify.ai,resources=difyclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dify.ai,resources=difyclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *DifyClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := r.Log.WithValues("difyCluster", req.NamespacedName)
    
    // 获取DifyCluster实例
    var difyCluster difyv1alpha1.DifyCluster
    if err := r.Get(ctx, req.NamespacedName, &difyCluster); err != nil {
        if errors.IsNotFound(err) {
            return ctrl.Result{}, nil
        }
        return ctrl.Result{}, err
    }
    
    // 处理删除逻辑
    if !difyCluster.DeletionTimestamp.IsZero() {
        return r.handleDeletion(ctx, &difyCluster)
    }
    
    // 添加Finalizer
    if !containsString(difyCluster.Finalizers, "dify.ai/finalizer") {
        difyCluster.Finalizers = append(difyCluster.Finalizers, "dify.ai/finalizer")
        return ctrl.Result{}, r.Update(ctx, &difyCluster)
    }
    
    // 协调各个组件
    if err := r.reconcileComponents(ctx, &difyCluster); err != nil {
        return ctrl.Result{RequeueAfter: time.Minute}, err
    }
    
    // 更新状态
    if err := r.updateStatus(ctx, &difyCluster); err != nil {
        return ctrl.Result{RequeueAfter: time.Minute}, err
    }
    
    return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}

func (r *DifyClusterReconciler) reconcileComponents(ctx context.Context, cluster *difyv1alpha1.DifyCluster) error {
    // 协调API组件
    if cluster.Spec.Components.API.Enabled {
        if err := r.reconcileAPIComponent(ctx, cluster); err != nil {
            return err
        }
    }
    
    // 协调Worker组件
    if cluster.Spec.Components.Worker.Enabled {
        if err := r.reconcileWorkerComponent(ctx, cluster); err != nil {
            return err
        }
    }
    
    // 协调Web组件
    if cluster.Spec.Components.Web.Enabled {
        if err := r.reconcileWebComponent(ctx, cluster); err != nil {
            return err
        }
    }
    
    // 协调Sandbox组件
    if cluster.Spec.Components.Sandbox.Enabled {
        if err := r.reconcileSandboxComponent(ctx, cluster); err != nil {
            return err
        }
    }
    
    return nil
}

func (r *DifyClusterReconciler) reconcileAPIComponent(ctx context.Context, cluster *difyv1alpha1.DifyCluster) error {
    // 构建API Deployment
    deployment := r.buildAPIDeployment(cluster)
    
    // 应用或更新Deployment
    if err := r.applyDeployment(ctx, deployment); err != nil {
        return err
    }
    
    // 构建API Service
    service := r.buildAPIService(cluster)
    
    // 应用或更新Service
    if err := r.applyService(ctx, service); err != nil {
        return err
    }
    
    // 如果启用了自动扩缩容，创建HPA
    if cluster.Spec.Components.API.Autoscaling.Enabled {
        hpa := r.buildAPIHPA(cluster)
        if err := r.applyHPA(ctx, hpa); err != nil {
            return err
        }
    }
    
    return nil
}

func (r *DifyClusterReconciler) buildAPIDeployment(cluster *difyv1alpha1.DifyCluster) *appsv1.Deployment {
    labels := map[string]string{
        "app.kubernetes.io/name":      "dify",
        "app.kubernetes.io/component": "api",
        "app.kubernetes.io/instance":  cluster.Name,
    }
    
    deployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("%s-api", cluster.Name),
            Namespace: cluster.Namespace,
            Labels:    labels,
            OwnerReferences: []metav1.OwnerReference{
                *metav1.NewControllerRef(cluster, difyv1alpha1.GroupVersion.WithKind("DifyCluster")),
            },
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: &cluster.Spec.Components.API.Replicas,
            Selector: &metav1.LabelSelector{
                MatchLabels: labels,
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: labels,
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:  "api",
                            Image: fmt.Sprintf("langgenius/dify-api:%s", cluster.Spec.DifyVersion),
                            Ports: []corev1.ContainerPort{
                                {
                                    ContainerPort: 5001,
                                    Protocol:      corev1.ProtocolTCP,
                                },
                            },
                            Resources: cluster.Spec.Components.API.Resources,
                            Env: r.buildAPIEnvVars(cluster),
                        },
                    },
                },
            },
        },
    }
    
    return deployment
}
```

#### 3.1.3 Helm Chart重构

**功能描述**：将原有Helm Chart重构为专门用于安装Operator和CRD

**重构后的Chart结构**：
```
dify-operator/
├── Chart.yaml
├── values.yaml
├── crds/
│   └── dify.ai_difyclusters.yaml      # DifyCluster CRD定义
├── templates/
│   ├── operator-deployment.yaml       # Operator部署配置
│   ├── operator-service.yaml          # Operator服务配置
│   ├── operator-rbac.yaml            # RBAC权限配置
│   ├── operator-configmap.yaml       # Operator配置
│   └── operator-serviceaccount.yaml  # ServiceAccount配置
└── templates/examples/
    └── difycruster-sample.yaml       # 示例DifyCluster配置
```

**新的values.yaml设计**：
```yaml
# dify-operator values.yaml
operator:
  image:
    repository: langgenius/dify-operator
    tag: "v0.1.0"
    pullPolicy: IfNotPresent
  
  replicas: 1
  
  resources:
    requests:
      cpu: "100m"
      memory: "128Mi"
    limits:
      cpu: "500m"
      memory: "512Mi"
  
  nodeSelector: {}
  tolerations: []
  affinity: {}
  
  # Operator配置
  config:
    logLevel: "info"
    metricsAddr: ":8080"
    probeAddr: ":8081"
    leaderElect: true
    
    # 默认镜像配置
    defaultImages:
      api: "langgenius/dify-api:1.7.1"
      web: "langgenius/dify-web:1.7.1"
      worker: "langgenius/dify-api:1.7.1"
      sandbox: "langgenius/dify-sandbox:0.2.12"
      pluginDaemon: "langgenius/dify-plugin-daemon:0.2.0-local"

# RBAC配置
rbac:
  create: true

serviceAccount:
  create: true
  name: "dify-operator"

# 监控配置
monitoring:
  enabled: false
  serviceMonitor:
    enabled: false

# 示例DifyCluster配置
examples:
  enabled: true
  difyCluster:
    name: "dify-sample"
    namespace: "dify-system"
```

### 3.2 迁移和兼容性设计

#### 3.2.1 配置迁移工具

**功能描述**：提供自动化工具将现有values.yaml配置转换为DifyCluster格式

**迁移脚本设计**：
```python
#!/usr/bin/env python3
# scripts/migrate-config.py

import yaml
import argparse
import sys
from pathlib import Path

class DifyConfigMigrator:
    def __init__(self, old_values_path, output_path):
        self.old_values_path = Path(old_values_path)
        self.output_path = Path(output_path)
        
    def migrate(self):
        """主迁移流程"""
        print("🔄 开始迁移Dify配置...")
        
        # 1. 读取旧配置
        old_config = self.load_old_config()
        
        # 2. 转换为新格式
        new_config = self.transform_config(old_config)
        
        # 3. 验证配置
        self.validate_config(new_config)
        
        # 4. 输出新配置
        self.save_new_config(new_config)
        
        print("✅ 配置迁移完成!")
        print(f"📄 新配置文件: {self.output_path}")
        
    def load_old_config(self):
        """读取旧的values.yaml配置"""
        if not self.old_values_path.exists():
            raise FileNotFoundError(f"配置文件不存在: {self.old_values_path}")
            
        with open(self.old_values_path, 'r', encoding='utf-8') as f:
            return yaml.safe_load(f)
    
    def transform_config(self, old_config):
        """转换配置格式"""
        new_config = {
            "apiVersion": "dify.ai/v1alpha1",
            "kind": "DifyCluster",
            "metadata": {
                "name": "dify-cluster",
                "namespace": "dify-system"
            },
            "spec": {
                "difyVersion": old_config.get("image", {}).get("api", {}).get("tag", "1.7.1"),
                "components": self.transform_components(old_config),
                "storage": self.transform_storage(old_config),
                "database": self.transform_database(old_config),
                "networking": self.transform_networking(old_config)
            }
        }
        
        return new_config
    
    def transform_components(self, old_config):
        """转换组件配置"""
        components = {}
        
        # API组件
        if "api" in old_config:
            api_config = old_config["api"]
            components["api"] = {
                "enabled": api_config.get("enabled", True),
                "replicas": api_config.get("replicas", 2),
                "autoscaling": {
                    "enabled": api_config.get("autoscaling", {}).get("enabled", True),
                    "minReplicas": api_config.get("autoscaling", {}).get("minReplicas", 1),
                    "maxReplicas": api_config.get("autoscaling", {}).get("maxReplicas", 20),
                    "targetCPUUtilization": api_config.get("autoscaling", {}).get("targetCPUUtilizationPercentage", 70)
                },
                "resources": api_config.get("resources", {
                    "requests": {"cpu": "500m", "memory": "1Gi"},
                    "limits": {"cpu": "2000m", "memory": "4Gi"}
                })
            }
        
        # Worker组件
        if "worker" in old_config:
            worker_config = old_config["worker"]
            components["worker"] = {
                "enabled": worker_config.get("enabled", True),
                "replicas": worker_config.get("replicas", 3),
                "autoscaling": {
                    "enabled": worker_config.get("autoscaling", {}).get("enabled", True),
                    "minReplicas": worker_config.get("autoscaling", {}).get("minReplicas", 2),
                    "maxReplicas": worker_config.get("autoscaling", {}).get("maxReplicas", 50)
                }
            }
        
        # Web组件  
        if "web" in old_config:
            web_config = old_config["web"]
            components["web"] = {
                "enabled": web_config.get("enabled", True),
                "replicas": web_config.get("replicas", 2)
            }
        
        # Sandbox组件
        if "sandbox" in old_config:
            sandbox_config = old_config["sandbox"]
            components["sandbox"] = {
                "enabled": sandbox_config.get("enabled", True),
                "replicas": sandbox_config.get("replicas", 1)
            }
        
        # Plugin Daemon组件
        if "pluginDaemon" in old_config:
            plugin_config = old_config["pluginDaemon"]
            components["pluginDaemon"] = {
                "enabled": plugin_config.get("enabled", True),
                "replicas": plugin_config.get("replicas", 1)
            }
        
        return components
    
    def transform_storage(self, old_config):
        """转换存储配置"""
        storage = {"type": "local"}
        
        # 检查S3配置
        if old_config.get("externalS3", {}).get("enabled", False):
            s3_config = old_config["externalS3"]
            storage = {
                "type": "s3",
                "s3": {
                    "endpoint": s3_config.get("endpoint", ""),
                    "bucket": s3_config.get("bucketName", {}).get("api", ""),
                    "accessKey": s3_config.get("accessKey", ""),
                    "secretKey": s3_config.get("secretKey", "")
                }
            }
        
        return storage
    
    def transform_database(self, old_config):
        """转换数据库配置"""
        database = {}
        
        # PostgreSQL配置
        if old_config.get("postgresql", {}).get("enabled", True):
            pg_config = old_config.get("postgresql", {})
            database["postgresql"] = {
                "enabled": True,
                "database": pg_config.get("auth", {}).get("database", "dify"),
                "username": pg_config.get("auth", {}).get("username", "postgres"),
                "password": pg_config.get("auth", {}).get("postgresPassword", "")
            }
        
        # Redis配置
        if old_config.get("redis", {}).get("enabled", True):
            redis_config = old_config.get("redis", {})
            database["redis"] = {
                "enabled": True,
                "password": redis_config.get("auth", {}).get("password", "")
            }
        
        return database
    
    def transform_networking(self, old_config):
        """转换网络配置"""
        networking = {}
        
        # Ingress配置
        if old_config.get("ingress", {}).get("enabled", False):
            ingress_config = old_config["ingress"]
            networking["ingress"] = {
                "enabled": True,
                "host": ingress_config.get("hosts", [{}])[0].get("host", ""),
                "tls": bool(ingress_config.get("tls", []))
            }
        
        # Service配置
        service_config = old_config.get("service", {})
        networking["service"] = {
            "type": service_config.get("type", "ClusterIP")
        }
        
        return networking
    
    def validate_config(self, config):
        """验证配置有效性"""
        required_fields = ["apiVersion", "kind", "metadata", "spec"]
        for field in required_fields:
            if field not in config:
                raise ValueError(f"必需字段缺失: {field}")
        
        print("✅ 配置验证通过")
    
    def save_new_config(self, config):
        """保存新配置"""
        self.output_path.parent.mkdir(parents=True, exist_ok=True)
        
        with open(self.output_path, 'w', encoding='utf-8') as f:
            yaml.dump(config, f, default_flow_style=False, allow_unicode=True, indent=2)

def main():
    parser = argparse.ArgumentParser(description="Dify配置迁移工具")
    parser.add_argument("--input", "-i", required=True, help="旧的values.yaml文件路径")
    parser.add_argument("--output", "-o", required=True, help="输出的DifyCluster配置文件路径")
    
    args = parser.parse_args()
    
    try:
        migrator = DifyConfigMigrator(args.input, args.output)
        migrator.migrate()
    except Exception as e:
        print(f"❌ 迁移失败: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
```

## 4. 技术实现方案

### 4.1 实现步骤

#### 阶段1：基础设施准备（第1-2周）

**Step 1.1: 创建Operator项目结构**
```bash
# 使用kubebuilder创建项目
kubebuilder init --domain dify.ai --repo github.com/langgenius/dify-operator

# 创建API
kubebuilder create api --group dify --version v1alpha1 --kind DifyCluster --resource --controller

# 生成CRD和RBAC
make manifests

# 项目结构
dify-operator/
├── api/
│   └── v1alpha1/
│       ├── difycruster_types.go    # CRD定义
│       └── groupversion_info.go
├── controllers/
│   └── difycruster_controller.go   # 控制器逻辑
├── config/
│   ├── crd/
│   │   └── bases/                  # 生成的CRD文件
│   ├── rbac/                       # RBAC配置
│   └── manager/                    # Manager配置
├── main.go                         # 主程序入口
├── Dockerfile                      # 容器化配置
└── Makefile                        # 构建脚本
```

**Step 1.2: 开发核心Controller逻辑**
```go
// api/v1alpha1/difycruster_types.go
package v1alpha1

import (
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DifyClusterSpec defines the desired state of DifyCluster
type DifyClusterSpec struct {
    // Dify版本
    DifyVersion string `json:"difyVersion,omitempty"`
    
    // 组件配置
    Components DifyComponents `json:"components,omitempty"`
    
    // 存储配置
    Storage DifyStorage `json:"storage,omitempty"`
    
    // 数据库配置
    Database DifyDatabase `json:"database,omitempty"`
    
    // 网络配置
    Networking DifyNetworking `json:"networking,omitempty"`
}

type DifyComponents struct {
    API          *APIComponent          `json:"api,omitempty"`
    Worker       *WorkerComponent       `json:"worker,omitempty"`
    Web          *WebComponent          `json:"web,omitempty"`
    Sandbox      *SandboxComponent      `json:"sandbox,omitempty"`
    PluginDaemon *PluginDaemonComponent `json:"pluginDaemon,omitempty"`
}

type APIComponent struct {
    Enabled     bool                     `json:"enabled,omitempty"`
    Replicas    int32                    `json:"replicas,omitempty"`
    Autoscaling *AutoscalingConfig       `json:"autoscaling,omitempty"`
    Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
}

type AutoscalingConfig struct {
    Enabled                bool  `json:"enabled,omitempty"`
    MinReplicas            int32 `json:"minReplicas,omitempty"`
    MaxReplicas            int32 `json:"maxReplicas,omitempty"`
    TargetCPUUtilization   int32 `json:"targetCPUUtilization,omitempty"`
}

// DifyClusterStatus defines the observed state of DifyCluster
type DifyClusterStatus struct {
    Phase            DifyClusterPhase    `json:"phase,omitempty"`
    Conditions       []metav1.Condition  `json:"conditions,omitempty"`
    ComponentStatus  ComponentStatus     `json:"componentStatus,omitempty"`
    ObservedGeneration int64             `json:"observedGeneration,omitempty"`
}

type DifyClusterPhase string

const (
    DifyClusterPhasePending   DifyClusterPhase = "Pending"
    DifyClusterPhaseRunning   DifyClusterPhase = "Running"
    DifyClusterPhaseUpgrading DifyClusterPhase = "Upgrading"
    DifyClusterPhaseFailed    DifyClusterPhase = "Failed"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Namespaced
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.difyVersion"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// DifyCluster is the Schema for the difyclusters API
type DifyCluster struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   DifyClusterSpec   `json:"spec,omitempty"`
    Status DifyClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DifyClusterList contains a list of DifyCluster
type DifyClusterList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []DifyCluster `json:"items"`
}

func init() {
    SchemeBuilder.Register(&DifyCluster{}, &DifyClusterList{})
}
```

#### 阶段2：Helm Chart重构（第3-4周）

**Step 2.1: 重构Chart结构**
```bash
# 备份原有Chart
cp -r charts/dify charts/dify-legacy

# 创建新的operator chart
mkdir -p charts/dify-operator/{templates,crds,examples}

# 迁移必要文件
cp charts/dify/Chart.yaml charts/dify-operator/
```

**Step 2.2: 创建Operator Deployment模板**
```yaml
# charts/dify-operator/templates/operator-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "dify-operator.fullname" . }}-controller-manager
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "dify-operator.labels" . | nindent 4 }}
    control-plane: controller-manager
spec:
  replicas: {{ .Values.operator.replicas }}
  selector:
    matchLabels:
      {{- include "dify-operator.selectorLabels" . | nindent 6 }}
      control-plane: controller-manager
  template:
    metadata:
      labels:
        {{- include "dify-operator.selectorLabels" . | nindent 8 }}
        control-plane: controller-manager
    spec:
      serviceAccountName: {{ include "dify-operator.serviceAccountName" . }}
      containers:
      - name: manager
        image: "{{ .Values.operator.image.repository }}:{{ .Values.operator.image.tag }}"
        imagePullPolicy: {{ .Values.operator.image.pullPolicy }}
        command:
        - /manager
        args:
        - --leader-elect
        - --metrics-bind-address=:{{ .Values.operator.config.metricsAddr | default "8080" }}
        - --health-probe-bind-address=:{{ .Values.operator.config.probeAddr | default "8081" }}
        env:
        - name: DIFY_DEFAULT_API_IMAGE
          value: "{{ .Values.operator.config.defaultImages.api }}"
        - name: DIFY_DEFAULT_WEB_IMAGE
          value: "{{ .Values.operator.config.defaultImages.web }}"
        - name: DIFY_DEFAULT_WORKER_IMAGE
          value: "{{ .Values.operator.config.defaultImages.worker }}"
        - name: DIFY_DEFAULT_SANDBOX_IMAGE
          value: "{{ .Values.operator.config.defaultImages.sandbox }}"
        ports:
        - containerPort: {{ .Values.operator.config.metricsAddr | default 8080 }}
          name: metrics
          protocol: TCP
        - containerPort: {{ .Values.operator.config.probeAddr | default 8081 }}
          name: health
          protocol: TCP
        livenessProbe:
          httpGet:
            path: /healthz
            port: health
          initialDelaySeconds: 15
          periodSeconds: 20
        readinessProbe:
          httpGet:
            path: /readyz
            port: health
          initialDelaySeconds: 5
          periodSeconds: 10
        resources:
          {{- toYaml .Values.operator.resources | nindent 10 }}
      {{- with .Values.operator.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.operator.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.operator.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
```

#### 阶段3：测试验证（第5周）

**Step 3.1: 单元测试**
```go
// controllers/difycruster_controller_test.go
package controllers

import (
    "context"
    "time"
    
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/resource"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/types"
    
    difyv1alpha1 "github.com/langgenius/dify-operator/api/v1alpha1"
)

var _ = Describe("DifyCluster Controller", func() {
    Context("When creating a DifyCluster", func() {
        It("Should create all required resources", func() {
            ctx := context.Background()
            
            // 创建测试DifyCluster
            difyCluster := &difyv1alpha1.DifyCluster{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-cluster",
                    Namespace: "default",
                },
                Spec: difyv1alpha1.DifyClusterSpec{
                    DifyVersion: "1.7.1",
                    Components: difyv1alpha1.DifyComponents{
                        API: &difyv1alpha1.APIComponent{
                            Enabled:  true,
                            Replicas: 2,
                            Resources: corev1.ResourceRequirements{
                                Requests: corev1.ResourceList{
                                    corev1.ResourceCPU:    resource.MustParse("500m"),
                                    corev1.ResourceMemory: resource.MustParse("1Gi"),
                                },
                            },
                        },
                    },
                },
            }
            
            Expect(k8sClient.Create(ctx, difyCluster)).Should(Succeed())
            
            // 验证API Deployment创建
            Eventually(func() bool {
                deployment := &appsv1.Deployment{}
                err := k8sClient.Get(ctx, types.NamespacedName{
                    Name:      "test-cluster-api",
                    Namespace: "default",
                }, deployment)
                return err == nil
            }, time.Second*10, time.Millisecond*250).Should(BeTrue())
            
            // 验证API Service创建
            Eventually(func() bool {
                service := &corev1.Service{}
                err := k8sClient.Get(ctx, types.NamespacedName{
                    Name:      "test-cluster-api",
                    Namespace: "default",
                }, service)
                return err == nil
            }, time.Second*10, time.Millisecond*250).Should(BeTrue())
        })
    })
    
    Context("When updating a DifyCluster", func() {
        It("Should update underlying resources", func() {
            ctx := context.Background()
            
            // 获取现有DifyCluster
            difyCluster := &difyv1alpha1.DifyCluster{}
            Expect(k8sClient.Get(ctx, types.NamespacedName{
                Name:      "test-cluster",
                Namespace: "default",
            }, difyCluster)).Should(Succeed())
            
            // 更新副本数
            difyCluster.Spec.Components.API.Replicas = 3
            Expect(k8sClient.Update(ctx, difyCluster)).Should(Succeed())
            
            // 验证Deployment更新
            Eventually(func() int32 {
                deployment := &appsv1.Deployment{}
                k8sClient.Get(ctx, types.NamespacedName{
                    Name:      "test-cluster-api",
                    Namespace: "default",
                }, deployment)
                return *deployment.Spec.Replicas
            }, time.Second*10, time.Millisecond*250).Should(Equal(int32(3)))
        })
    })
})
```

### 4.2 配置相关

#### 4.2.1 构建和部署配置

**Dockerfile**:
```dockerfile
# Build the manager binary
FROM golang:1.21 as builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
```

**Makefile扩展**:
```makefile
# Build manager binary
.PHONY: build
build: manifests generate fmt vet
	go build -o bin/manager main.go

# Build docker image
.PHONY: docker-build
docker-build:
	docker build -t langgenius/dify-operator:$(VERSION) .

# Push docker image
.PHONY: docker-push
docker-push:
	docker push langgenius/dify-operator:$(VERSION)

# Generate helm chart
.PHONY: helm-package
helm-package: manifests
	# Copy CRD to helm chart
	cp config/crd/bases/dify.ai_difyclusters.yaml charts/dify-operator/crds/
	# Package helm chart
	helm package charts/dify-operator -d dist/

# Deploy operator using helm
.PHONY: helm-install
helm-install: helm-package
	helm upgrade --install dify-operator dist/dify-operator-*.tgz \
		--namespace dify-operator-system --create-namespace

# Generate sample DifyCluster
.PHONY: generate-sample
generate-sample:
	cat > config/samples/dify_v1alpha1_difycruster.yaml << 'EOF'
	apiVersion: dify.ai/v1alpha1
	kind: DifyCluster
	metadata:
	  name: dify-sample
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
	      resources:
	        requests:
	          cpu: "500m"
	          memory: "1Gi"
	        limits:
	          cpu: "2000m"
	          memory: "4Gi"
	    worker:
	      enabled: true
	      replicas: 3
	      autoscaling:
	        enabled: true
	        minReplicas: 2
	        maxReplicas: 20
	    web:
	      enabled: true
	      replicas: 2
	    sandbox:
	      enabled: true
	      replicas: 1
	    pluginDaemon:
	      enabled: true
	      replicas: 1
	  storage:
	    type: "local"
	  database:
	    postgresql:
	      enabled: true
	      database: "dify"
	      username: "postgres"
	      password: "dify123456"
	    redis:
	      enabled: true
	      password: "dify123456"
	  networking:
	    service:
	      type: "ClusterIP"
	EOF
```

### 4.3 测试验证

#### 4.3.1 端到端测试脚本

```bash
#!/bin/bash
# scripts/e2e-test.sh

set -e

echo "🚀 开始Dify Operator端到端测试..."

# 1. 准备测试环境
echo "📋 准备测试环境..."
kubectl create namespace dify-test --dry-run=client -o yaml | kubectl apply -f -

# 2. 安装Operator
echo "📦 安装Dify Operator..."
helm upgrade --install dify-operator charts/dify-operator \
  --namespace dify-operator-system --create-namespace \
  --wait --timeout=300s

# 3. 等待Operator就绪
echo "⏳ 等待Operator就绪..."
kubectl wait --for=condition=available deployment/dify-operator-controller-manager \
  -n dify-operator-system --timeout=300s

# 4. 创建测试DifyCluster
echo "🏗️ 创建测试DifyCluster..."
cat << EOF | kubectl apply -f -
apiVersion: dify.ai/v1alpha1
kind: DifyCluster
metadata:
  name: test-cluster
  namespace: dify-test
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
    web:
      enabled: true
      replicas: 1
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
      password: "test123456"
    redis:
      enabled: true
      password: "test123456"
EOF

# 5. 等待DifyCluster就绪
echo "⏳ 等待DifyCluster就绪..."
timeout=300
while [ $timeout -gt 0 ]; do
  phase=$(kubectl get difycruster test-cluster -n dify-test -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
  if [ "$phase" = "Running" ]; then
    echo "✅ DifyCluster已就绪"
    break
  fi
  echo "⏳ 当前状态: $phase, 等待中..."
  sleep 10
  timeout=$((timeout - 10))
done

if [ $timeout -le 0 ]; then
  echo "❌ DifyCluster启动超时"
  kubectl describe difycruster test-cluster -n dify-test
  exit 1
fi

# 6. 验证组件状态
echo "🔍 验证组件状态..."
components=("api" "worker" "web" "sandbox")
for component in "${components[@]}"; do
  echo "  检查 $component 组件..."
  kubectl wait --for=condition=available deployment/test-cluster-$component \
    -n dify-test --timeout=300s
  echo "  ✅ $component 组件正常"
done

# 7. 测试扩缩容
echo "📈 测试扩缩容功能..."
kubectl patch difycruster test-cluster -n dify-test --type='merge' \
  -p='{"spec":{"components":{"api":{"replicas":2}}}}'

echo "⏳ 等待API组件扩容..."
kubectl wait --for=jsonpath='{.spec.replicas}'=2 deployment/test-cluster-api \
  -n dify-test --timeout=300s
echo "✅ 扩缩容测试通过"

# 8. 测试版本升级
echo "🔄 测试版本升级..."
kubectl patch difycruster test-cluster -n dify-test --type='merge' \
  -p='{"spec":{"difyVersion":"1.7.1"}}'

echo "⏳ 等待版本升级..."
timeout=300
while [ $timeout -gt 0 ]; do
  phase=$(kubectl get difycruster test-cluster -n dify-test -o jsonpath='{.status.phase}')
  if [ "$phase" = "Running" ]; then
    echo "✅ 版本升级完成"
    break
  fi
  echo "⏳ 升级状态: $phase"
  sleep 10
  timeout=$((timeout - 10))
done

# 9. 健康检查
echo "🏥 执行健康检查..."
api_pod=$(kubectl get pods -n dify-test -l app.kubernetes.io/component=api -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n dify-test $api_pod -- wget -q --spider http://localhost:5001/health

echo "✅ 健康检查通过"

# 10. 清理测试资源
echo "🧹 清理测试资源..."
kubectl delete difycruster test-cluster -n dify-test
kubectl delete namespace dify-test

echo "🎉 端到端测试完成!"
```

## 5. 质量保证

### 5.1 测试策略

#### 5.1.1 功能测试清单

**CRD功能测试**：
- [ ] DifyCluster资源创建、更新、删除
- [ ] 字段验证和默认值设置
- [ ] Status字段更新机制
- [ ] Owner Reference正确设置

**Controller功能测试**：
- [ ] 组件Deployment创建和更新
- [ ] Service和ConfigMap管理
- [ ] HPA自动扩缩容配置
- [ ] 错误处理和重试机制

**Helm Chart测试**：
- [ ] Chart安装和卸载
- [ ] Values配置验证
- [ ] 模板渲染正确性
- [ ] RBAC权限配置

#### 5.1.2 性能测试

**负载测试配置**：
```yaml
# tests/performance/load-test.yaml
apiVersion: dify.ai/v1alpha1
kind: DifyCluster
metadata:
  name: perf-test
  namespace: dify-perf
spec:
  difyVersion: "1.7.1"
  components:
    api:
      enabled: true
      replicas: 5
      autoscaling:
        enabled: true
        minReplicas: 3
        maxReplicas: 50
        targetCPUUtilization: 60
      resources:
        requests:
          cpu: "1000m"
          memory: "2Gi"
        limits:
          cpu: "4000m"
          memory: "8Gi"
    worker:
      enabled: true
      replicas: 10
      autoscaling:
        enabled: true
        minReplicas: 5
        maxReplicas: 100
```

### 5.2 风险控制

#### 5.2.1 升级风险控制

**渐进式升级策略**：
```yaml
# 升级策略配置
upgrade:
  strategy: "RollingUpdate"
  rollingUpdate:
    maxUnavailable: 1
    maxSurge: 1
  
  # 金丝雀发布
  canary:
    enabled: true
    steps:
    - weight: 10
      pause: "5m"
    - weight: 30
      pause: "10m"
    - weight: 50
      pause: "15m"
    - weight: 100
```

**回滚机制**：
```bash
# 自动回滚脚本
#!/bin/bash
# scripts/auto-rollback.sh

CLUSTER_NAME=$1
NAMESPACE=$2
TIMEOUT=${3:-600}

echo "监控集群 $CLUSTER_NAME 升级状态..."

start_time=$(date +%s)
while true; do
  current_time=$(date +%s)
  elapsed=$((current_time - start_time))
  
  if [ $elapsed -gt $TIMEOUT ]; then
    echo "❌ 升级超时，开始自动回滚..."
    kubectl patch difycruster $CLUSTER_NAME -n $NAMESPACE --type='merge' \
      -p='{"spec":{"difyVersion":"1.7.1"}}'
    break
  fi
  
  phase=$(kubectl get difycruster $CLUSTER_NAME -n $NAMESPACE -o jsonpath='{.status.phase}')
  
  if [ "$phase" = "Failed" ]; then
    echo "❌ 升级失败，开始自动回滚..."
    kubectl patch difycruster $CLUSTER_NAME -n $NAMESPACE --type='merge' \
      -p='{"spec":{"difyVersion":"1.7.1"}}'
    break
  elif [ "$phase" = "Running" ]; then
    echo "✅ 升级成功完成"
    break
  fi
  
  echo "⏳ 当前状态: $phase"
  sleep 30
done
```

## 6. 上线计划

### 6.1 开发阶段

#### Sprint 1 (第1-2周): 基础设施开发
**目标**: 完成Operator基础框架和CRD定义

**任务清单**:
- [ ] 使用kubebuilder创建项目结构
- [ ] 定义DifyCluster CRD Schema
- [ ] 实现基础Controller框架
- [ ] 编写单元测试用例
- [ ] 配置CI/CD流水线

**交付物**:
- [ ] 完整的Operator项目代码
- [ ] CRD定义文件
- [ ] 基础单元测试

#### Sprint 2 (第3-4周): 核心功能实现
**目标**: 实现组件管理和生命周期控制

**任务清单**:
- [ ] 实现API组件Controller逻辑
- [ ] 实现Worker组件Controller逻辑
- [ ] 实现Web、Sandbox、PluginDaemon组件管理
- [ ] 实现HPA自动扩缩容功能
- [ ] 开发配置迁移工具

**交付物**:
- [ ] 完整的组件管理功能
- [ ] 自动扩缩容实现
- [ ] 配置迁移脚本

#### Sprint 3 (第5周): Helm Chart重构
**目标**: 重构Helm Chart为Operator模式

**任务清单**:
- [ ] 重构Chart目录结构
- [ ] 创建Operator部署模板
- [ ] 编写示例DifyCluster配置
- [ ] 更新文档和使用指南

**交付物**:
- [ ] 新的dify-operator Helm Chart
- [ ] 部署文档和示例
- [ ] 迁移指南

#### Sprint 4 (第6周): 测试和优化
**目标**: 完成全面测试和性能优化

**任务清单**:
- [ ] 端到端测试验证
- [ ] 性能测试和优化
- [ ] 安全性测试
- [ ] 文档完善

**交付物**:
- [ ] 测试报告
- [ ] 性能基准
- [ ] 完整文档

### 6.2 验收标准

#### 6.2.1 功能验收标准

**基础功能验收**:
- [ ] DifyCluster CRD成功安装并可创建实例
- [ ] Operator能够监听CRD变化并创建相应资源
- [ ] 所有组件(API、Worker、Web、Sandbox、PluginDaemon)正常部署
- [ ] 自动扩缩容功能正常工作
- [ ] 配置更新能够触发资源更新

**高级功能验收**:
- [ ] 版本升级功能正常，支持滚动更新
- [ ] 故障自愈功能正常，Pod重启后自动恢复
- [ ] 监控和日志功能集成正常
- [ ] RBAC权限配置正确，符合最小权限原则

#### 6.2.2 性能验收标准

**响应性能**:
- [ ] DifyCluster创建到Ready状态 < 5分钟
- [ ] 配置更新响应时间 < 30秒
- [ ] 自动扩缩容响应时间 < 2分钟
- [ ] 版本升级完成时间 < 10分钟

**资源效率**:
- [ ] Operator内存使用 < 512MB
- [ ] Operator CPU使用率 < 5%（空闲时）
- [ ] 整体资源利用率提升 > 20%

#### 6.2.3 稳定性验收标准

**可靠性测试**:
- [ ] 7x24小时稳定性测试，无异常重启
- [ ] 网络分区测试，能够正确处理网络故障
- [ ] 节点故障测试，Pod能够自动重新调度
- [ ] 大规模测试，支持100+ Pod的集群管理

**兼容性测试**:
- [ ] Kubernetes 1.18+ 版本兼容
- [ ] 多种存储后端兼容性验证
- [ ] 不同云厂商环境验证

## 7. 迁移执行计划

### 7.1 迁移策略

#### 7.1.1 蓝绿部署迁移
**策略说明**: 在现有环境旁部署新的Operator环境，验证无误后切换

**执行步骤**:
```bash
# 1. 在新namespace部署operator环境
kubectl create namespace dify-operator-blue

# 2. 安装dify-operator
helm install dify-operator-blue charts/dify-operator \
  --namespace dify-operator-blue

# 3. 迁移配置并部署新集群
python3 scripts/migrate-config.py \
  --input charts/dify/values.yaml \
  --output dify-cluster-blue.yaml

kubectl apply -f dify-cluster-blue.yaml

# 4. 验证新环境
./scripts/e2e-test.sh dify-cluster-blue dify-system-blue

# 5. 切换流量（通过ingress或service）
kubectl patch ingress dify-ingress --type='merge' \
  -p='{"spec":{"rules":[{"http":{"paths":[{"backend":{"service":{"name":"dify-cluster-blue-proxy"}}}]}}]}}'

# 6. 清理旧环境
helm uninstall dify-legacy
```

### 7.2 应急预案

#### 7.2.1 快速回滚方案
```bash
#!/bin/bash
# scripts/emergency-rollback.sh

echo "🚨 执行紧急回滚..."

# 1. 切换回旧版本服务
kubectl patch ingress dify-ingress --type='merge' \
  -p='{"spec":{"rules":[{"http":{"paths":[{"backend":{"service":{"name":"dify-legacy-proxy"}}}]}}]}}'

# 2. 重新部署旧版本（如果已删除）
if ! helm list | grep -q dify-legacy; then
  helm install dify-legacy charts/dify-legacy -f backup-values.yaml
fi

# 3. 验证旧版本服务
curl -f http://dify.example.com/health

echo "✅ 回滚完成"
```

## 总结

本PRD定义了将dify-helm从传统Helm模式升级到Operator模式的完整方案。通过引入DifyCluster CRD和Operator控制器，我们将实现：

1. **配置简化**: 从3400行配置降低到1000行以内
2. **运维自动化**: 故障自愈、智能扩缩容、版本管理
3. **架构现代化**: 符合Cloud Native最佳实践
4. **向后兼容**: 平滑迁移现有部署

升级完成后，用户只需要创建一个DifyCluster资源即可完成整个Dify系统的部署和管理，大大简化了运维复杂度，提升了系统的可靠性和可扩展性。
