#!/usr/bin/env python3
"""
Dify配置迁移工具
将旧的values.yaml配置转换为DifyCluster格式
"""

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
        print(f"📋 使用方式:")
        print(f"   kubectl apply -f {self.output_path}")
        
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
                "difyVersion": self.extract_version(old_config),
                "components": self.transform_components(old_config),
                "storage": self.transform_storage(old_config),
                "database": self.transform_database(old_config),
                "networking": self.transform_networking(old_config)
            }
        }
        
        return new_config
    
    def extract_version(self, old_config):
        """提取Dify版本"""
        # 从API镜像标签中提取版本
        api_image = old_config.get("image", {}).get("api", {})
        version = api_image.get("tag", "1.7.1")
        
        print(f"📋 检测到Dify版本: {version}")
        return version
    
    def transform_components(self, old_config):
        """转换组件配置"""
        components = {}
        
        # API组件
        if "api" in old_config:
            api_config = old_config["api"]
            components["api"] = {
                "enabled": api_config.get("enabled", True),
                "replicas": api_config.get("replicas", 2),
                "autoscaling": self.transform_autoscaling(api_config.get("autoscaling", {})),
                "resources": self.transform_resources(api_config.get("resources", {}))
            }
            print(f"✅ 转换API组件配置: {components['api']['replicas']} 个副本")
        
        # Worker组件
        if "worker" in old_config:
            worker_config = old_config["worker"]
            components["worker"] = {
                "enabled": worker_config.get("enabled", True),
                "replicas": worker_config.get("replicas", 3),
                "autoscaling": self.transform_autoscaling(worker_config.get("autoscaling", {})),
                "resources": self.transform_resources(worker_config.get("resources", {}))
            }
            print(f"✅ 转换Worker组件配置: {components['worker']['replicas']} 个副本")
        
        # Web组件  
        if "web" in old_config:
            web_config = old_config["web"]
            components["web"] = {
                "enabled": web_config.get("enabled", True),
                "replicas": web_config.get("replicas", 2),
                "resources": self.transform_resources(web_config.get("resources", {}))
            }
            print(f"✅ 转换Web组件配置: {components['web']['replicas']} 个副本")
        
        # Sandbox组件
        if "sandbox" in old_config:
            sandbox_config = old_config["sandbox"]
            components["sandbox"] = {
                "enabled": sandbox_config.get("enabled", True),
                "replicas": sandbox_config.get("replicas", 1),
                "autoscaling": self.transform_autoscaling(sandbox_config.get("autoscaling", {})),
                "resources": self.transform_resources(sandbox_config.get("resources", {}))
            }
            print(f"✅ 转换Sandbox组件配置: {components['sandbox']['replicas']} 个副本")
        
        # Plugin Daemon组件
        if "pluginDaemon" in old_config:
            plugin_config = old_config["pluginDaemon"]
            components["pluginDaemon"] = {
                "enabled": plugin_config.get("enabled", True),
                "replicas": plugin_config.get("replicas", 1),
                "resources": self.transform_resources(plugin_config.get("resources", {}))
            }
            print(f"✅ 转换PluginDaemon组件配置: {components['pluginDaemon']['replicas']} 个副本")
        
        return components
    
    def transform_autoscaling(self, autoscaling_config):
        """转换自动扩缩容配置"""
        if not autoscaling_config.get("enabled", False):
            return None
            
        return {
            "enabled": True,
            "minReplicas": autoscaling_config.get("minReplicas", 1),
            "maxReplicas": autoscaling_config.get("maxReplicas", 10),
            "targetCPUUtilization": autoscaling_config.get("targetCPUUtilizationPercentage", 70),
            "targetMemoryUtilization": autoscaling_config.get("targetMemoryUtilizationPercentage", 80)
        }
    
    def transform_resources(self, resources_config):
        """转换资源配置"""
        if not resources_config:
            return {
                "requests": {"cpu": "100m", "memory": "256Mi"},
                "limits": {"cpu": "500m", "memory": "1Gi"}
            }
        
        return {
            "requests": resources_config.get("requests", {"cpu": "100m", "memory": "256Mi"}),
            "limits": resources_config.get("limits", {"cpu": "500m", "memory": "1Gi"})
        }
    
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
                    "secretKey": s3_config.get("secretKey", ""),
                    "region": s3_config.get("region", "us-east-1")
                }
            }
            print(f"✅ 转换S3存储配置: {storage['s3']['bucket']}")
        
        # 检查GCS配置
        elif old_config.get("externalGCS", {}).get("enabled", False):
            gcs_config = old_config["externalGCS"]
            storage = {
                "type": "gcs",
                "gcs": {
                    "bucket": gcs_config.get("bucketName", {}).get("api", ""),
                    "serviceAccountJsonBase64": gcs_config.get("serviceAccountJsonBase64", "")
                }
            }
            print(f"✅ 转换GCS存储配置: {storage['gcs']['bucket']}")
        
        else:
            print("✅ 使用本地存储配置")
        
        return storage
    
    def transform_database(self, old_config):
        """转换数据库配置"""
        database = {}
        
        # PostgreSQL配置
        if old_config.get("postgresql", {}).get("enabled", True):
            pg_config = old_config.get("postgresql", {})
            auth_config = pg_config.get("auth", {})
            
            database["postgresql"] = {
                "enabled": True,
                "host": old_config.get("externalDatabase", {}).get("host", "dify-postgresql"),
                "database": auth_config.get("database", "dify"),
                "username": auth_config.get("username", "postgres"),
                "password": auth_config.get("postgresPassword", "dify123456")
            }
            print(f"✅ 转换PostgreSQL配置: {database['postgresql']['database']}")
        
        # Redis配置
        if old_config.get("redis", {}).get("enabled", True):
            redis_config = old_config.get("redis", {})
            auth_config = redis_config.get("auth", {})
            
            database["redis"] = {
                "enabled": True,
                "host": old_config.get("externalRedis", {}).get("host", "dify-redis"),
                "password": auth_config.get("password", "dify123456")
            }
            print(f"✅ 转换Redis配置")
        
        return database
    
    def transform_networking(self, old_config):
        """转换网络配置"""
        networking = {}
        
        # Ingress配置
        if old_config.get("ingress", {}).get("enabled", False):
            ingress_config = old_config["ingress"]
            hosts = ingress_config.get("hosts", [])
            
            networking["ingress"] = {
                "enabled": True,
                "host": hosts[0].get("host", "dify.example.com") if hosts else "dify.example.com",
                "tls": bool(ingress_config.get("tls", []))
            }
            print(f"✅ 转换Ingress配置: {networking['ingress']['host']}")
        
        # Service配置
        service_config = old_config.get("service", {})
        networking["service"] = {
            "type": service_config.get("type", "ClusterIP")
        }
        print(f"✅ 转换Service配置: {networking['service']['type']}")
        
        return networking
    
    def validate_config(self, config):
        """验证配置有效性"""
        required_fields = ["apiVersion", "kind", "metadata", "spec"]
        for field in required_fields:
            if field not in config:
                raise ValueError(f"必需字段缺失: {field}")
        
        # 验证组件配置
        components = config["spec"].get("components", {})
        if not any(comp.get("enabled", False) for comp in components.values()):
            raise ValueError("至少需要启用一个组件")
        
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
    parser.add_argument("--name", "-n", default="dify-cluster", help="DifyCluster名称")
    parser.add_argument("--namespace", "-ns", default="dify-system", help="目标命名空间")
    
    args = parser.parse_args()
    
    try:
        migrator = DifyConfigMigrator(args.input, args.output)
        migrator.migrate()
        
        print(f"\n🎉 迁移成功完成!")
        print(f"🚀 接下来的步骤:")
        print(f"   1. 安装dify-operator:")
        print(f"      helm install dify-operator charts/dify-operator")
        print(f"   2. 部署DifyCluster:")
        print(f"      kubectl apply -f {args.output}")
        print(f"   3. 检查状态:")
        print(f"      kubectl get difycruster -n {args.namespace}")
        
    except Exception as e:
        print(f"❌ 迁移失败: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()