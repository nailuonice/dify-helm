/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json:"-" or json:"foo".

// DifyClusterSpec defines the desired state of DifyCluster
type DifyClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

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

// DifyComponents defines the components configuration
type DifyComponents struct {
	API          *APIComponent          `json:"api,omitempty"`
	Worker       *WorkerComponent       `json:"worker,omitempty"`
	Web          *WebComponent          `json:"web,omitempty"`
	Sandbox      *SandboxComponent      `json:"sandbox,omitempty"`
	PluginDaemon *PluginDaemonComponent `json:"pluginDaemon,omitempty"`
}

// APIComponent defines the API component configuration
type APIComponent struct {
	Enabled     bool                        `json:"enabled,omitempty"`
	Replicas    int32                       `json:"replicas,omitempty"`
	Autoscaling *AutoscalingConfig          `json:"autoscaling,omitempty"`
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
}

// WorkerComponent defines the Worker component configuration
type WorkerComponent struct {
	Enabled     bool                        `json:"enabled,omitempty"`
	Replicas    int32                       `json:"replicas,omitempty"`
	Autoscaling *AutoscalingConfig          `json:"autoscaling,omitempty"`
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
}

// WebComponent defines the Web component configuration
type WebComponent struct {
	Enabled     bool                        `json:"enabled,omitempty"`
	Replicas    int32                       `json:"replicas,omitempty"`
	Autoscaling *AutoscalingConfig          `json:"autoscaling,omitempty"`
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
}

// SandboxComponent defines the Sandbox component configuration
type SandboxComponent struct {
	Enabled     bool                        `json:"enabled,omitempty"`
	Replicas    int32                       `json:"replicas,omitempty"`
	Autoscaling *AutoscalingConfig          `json:"autoscaling,omitempty"`
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
}

// PluginDaemonComponent defines the PluginDaemon component configuration
type PluginDaemonComponent struct {
	Enabled   bool                        `json:"enabled,omitempty"`
	Replicas  int32                       `json:"replicas,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// AutoscalingConfig defines the autoscaling configuration
type AutoscalingConfig struct {
	Enabled                bool  `json:"enabled,omitempty"`
	MinReplicas            int32 `json:"minReplicas,omitempty"`
	MaxReplicas            int32 `json:"maxReplicas,omitempty"`
	TargetCPUUtilization   int32 `json:"targetCPUUtilization,omitempty"`
	TargetMemoryUtilization int32 `json:"targetMemoryUtilization,omitempty"`
}

// DifyStorage defines the storage configuration
type DifyStorage struct {
	Type string    `json:"type,omitempty"`
	S3   *S3Config `json:"s3,omitempty"`
	GCS  *GCSConfig `json:"gcs,omitempty"`
}

// S3Config defines the S3 storage configuration
type S3Config struct {
	Endpoint  string `json:"endpoint,omitempty"`
	Bucket    string `json:"bucket,omitempty"`
	AccessKey string `json:"accessKey,omitempty"`
	SecretKey string `json:"secretKey,omitempty"`
	Region    string `json:"region,omitempty"`
}

// GCSConfig defines the GCS storage configuration
type GCSConfig struct {
	Bucket                    string `json:"bucket,omitempty"`
	ServiceAccountJsonBase64  string `json:"serviceAccountJsonBase64,omitempty"`
}

// DifyDatabase defines the database configuration
type DifyDatabase struct {
	PostgreSQL *PostgreSQLConfig `json:"postgresql,omitempty"`
	Redis      *RedisConfig      `json:"redis,omitempty"`
}

// PostgreSQLConfig defines the PostgreSQL configuration
type PostgreSQLConfig struct {
	Enabled  bool   `json:"enabled,omitempty"`
	Host     string `json:"host,omitempty"`
	Database string `json:"database,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// RedisConfig defines the Redis configuration
type RedisConfig struct {
	Enabled  bool   `json:"enabled,omitempty"`
	Host     string `json:"host,omitempty"`
	Password string `json:"password,omitempty"`
}

// DifyNetworking defines the networking configuration
type DifyNetworking struct {
	Ingress *IngressConfig `json:"ingress,omitempty"`
	Service *ServiceConfig `json:"service,omitempty"`
}

// IngressConfig defines the ingress configuration
type IngressConfig struct {
	Enabled bool   `json:"enabled,omitempty"`
	Host    string `json:"host,omitempty"`
	TLS     bool   `json:"tls,omitempty"`
}

// ServiceConfig defines the service configuration
type ServiceConfig struct {
	Type string `json:"type,omitempty"`
}

// DifyClusterStatus defines the observed state of DifyCluster
type DifyClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Phase              DifyClusterPhase `json:"phase,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	ComponentStatus    ComponentStatus  `json:"componentStatus,omitempty"`
	ObservedGeneration int64            `json:"observedGeneration,omitempty"`
}

// DifyClusterPhase defines the phase of DifyCluster
type DifyClusterPhase string

const (
	DifyClusterPhasePending   DifyClusterPhase = "Pending"
	DifyClusterPhaseRunning   DifyClusterPhase = "Running"
	DifyClusterPhaseUpgrading DifyClusterPhase = "Upgrading"
	DifyClusterPhaseFailed    DifyClusterPhase = "Failed"
)

// ComponentStatus defines the status of each component
type ComponentStatus struct {
	API          *ComponentInfo `json:"api,omitempty"`
	Worker       *ComponentInfo `json:"worker,omitempty"`
	Web          *ComponentInfo `json:"web,omitempty"`
	Sandbox      *ComponentInfo `json:"sandbox,omitempty"`
	PluginDaemon *ComponentInfo `json:"pluginDaemon,omitempty"`
}

// ComponentInfo defines the information of a component
type ComponentInfo struct {
	Ready         bool  `json:"ready,omitempty"`
	Replicas      int32 `json:"replicas,omitempty"`
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`
}

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