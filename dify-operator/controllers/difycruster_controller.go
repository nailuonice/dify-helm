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

package controllers

import (
	"context"
	"fmt"
	"os"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	difyv1alpha1 "github.com/langgenius/dify-operator/api/v1alpha1"
)

// DifyClusterReconciler reconciles a DifyCluster object
type DifyClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=dify.ai,resources=difyclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dify.ai,resources=difyclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dify.ai,resources=difyclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DifyClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 获取DifyCluster实例
	var difyCluster difyv1alpha1.DifyCluster
	if err := r.Get(ctx, req.NamespacedName, &difyCluster); err != nil {
		if errors.IsNotFound(err) {
			// 对象被删除，返回并不重新排队
			logger.Info("DifyCluster resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get DifyCluster")
		return ctrl.Result{}, err
	}

	// 处理删除逻辑
	if !difyCluster.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &difyCluster)
	}

	// 添加Finalizer
	if !containsString(difyCluster.Finalizers, "dify.ai/finalizer") {
		difyCluster.Finalizers = append(difyCluster.Finalizers, "dify.ai/finalizer")
		if err := r.Update(ctx, &difyCluster); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// 协调各个组件
	if err := r.reconcileComponents(ctx, &difyCluster); err != nil {
		logger.Error(err, "Failed to reconcile components")
		r.updateStatus(ctx, &difyCluster, difyv1alpha1.DifyClusterPhaseFailed)
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// 更新状态
	if err := r.updateStatus(ctx, &difyCluster, difyv1alpha1.DifyClusterPhaseRunning); err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}

// handleDeletion handles the deletion of DifyCluster
func (r *DifyClusterReconciler) handleDeletion(ctx context.Context, cluster *difyv1alpha1.DifyCluster) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling DifyCluster deletion")

	// 清理资源的逻辑在这里
	// Kubernetes会自动清理OwnerReference设置的资源

	// 移除finalizer
	cluster.Finalizers = removeString(cluster.Finalizers, "dify.ai/finalizer")
	if err := r.Update(ctx, cluster); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// reconcileComponents reconciles all components
func (r *DifyClusterReconciler) reconcileComponents(ctx context.Context, cluster *difyv1alpha1.DifyCluster) error {
	// 协调API组件
	if cluster.Spec.Components.API != nil && cluster.Spec.Components.API.Enabled {
		if err := r.reconcileAPIComponent(ctx, cluster); err != nil {
			return err
		}
	}

	// 协调Worker组件
	if cluster.Spec.Components.Worker != nil && cluster.Spec.Components.Worker.Enabled {
		if err := r.reconcileWorkerComponent(ctx, cluster); err != nil {
			return err
		}
	}

	// 协调Web组件
	if cluster.Spec.Components.Web != nil && cluster.Spec.Components.Web.Enabled {
		if err := r.reconcileWebComponent(ctx, cluster); err != nil {
			return err
		}
	}

	// 协调Sandbox组件
	if cluster.Spec.Components.Sandbox != nil && cluster.Spec.Components.Sandbox.Enabled {
		if err := r.reconcileSandboxComponent(ctx, cluster); err != nil {
			return err
		}
	}

	return nil
}

// reconcileAPIComponent reconciles the API component
func (r *DifyClusterReconciler) reconcileAPIComponent(ctx context.Context, cluster *difyv1alpha1.DifyCluster) error {
	// 构建API Deployment
	deployment := r.buildAPIDeployment(cluster)
	if err := r.applyDeployment(ctx, deployment); err != nil {
		return err
	}

	// 构建API Service
	service := r.buildAPIService(cluster)
	if err := r.applyService(ctx, service); err != nil {
		return err
	}

	// 如果启用了自动扩缩容，创建HPA
	if cluster.Spec.Components.API.Autoscaling != nil && cluster.Spec.Components.API.Autoscaling.Enabled {
		hpa := r.buildAPIHPA(cluster)
		if err := r.applyHPA(ctx, hpa); err != nil {
			return err
		}
	}

	return nil
}

// reconcileWorkerComponent reconciles the Worker component
func (r *DifyClusterReconciler) reconcileWorkerComponent(ctx context.Context, cluster *difyv1alpha1.DifyCluster) error {
	// 构建Worker Deployment
	deployment := r.buildWorkerDeployment(cluster)
	if err := r.applyDeployment(ctx, deployment); err != nil {
		return err
	}

	// 如果启用了自动扩缩容，创建HPA
	if cluster.Spec.Components.Worker.Autoscaling != nil && cluster.Spec.Components.Worker.Autoscaling.Enabled {
		hpa := r.buildWorkerHPA(cluster)
		if err := r.applyHPA(ctx, hpa); err != nil {
			return err
		}
	}

	return nil
}

// reconcileWebComponent reconciles the Web component
func (r *DifyClusterReconciler) reconcileWebComponent(ctx context.Context, cluster *difyv1alpha1.DifyCluster) error {
	// 构建Web Deployment
	deployment := r.buildWebDeployment(cluster)
	if err := r.applyDeployment(ctx, deployment); err != nil {
		return err
	}

	// 构建Web Service
	service := r.buildWebService(cluster)
	if err := r.applyService(ctx, service); err != nil {
		return err
	}

	return nil
}

// reconcileSandboxComponent reconciles the Sandbox component
func (r *DifyClusterReconciler) reconcileSandboxComponent(ctx context.Context, cluster *difyv1alpha1.DifyCluster) error {
	// 构建Sandbox Deployment
	deployment := r.buildSandboxDeployment(cluster)
	if err := r.applyDeployment(ctx, deployment); err != nil {
		return err
	}

	// 构建Sandbox Service
	service := r.buildSandboxService(cluster)
	if err := r.applyService(ctx, service); err != nil {
		return err
	}

	return nil
}

// buildAPIDeployment builds the API deployment
func (r *DifyClusterReconciler) buildAPIDeployment(cluster *difyv1alpha1.DifyCluster) *appsv1.Deployment {
	labels := r.buildLabels(cluster.Name, "api")
	replicas := cluster.Spec.Components.API.Replicas
	if replicas == 0 {
		replicas = 2 // 默认值
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
			Replicas: &replicas,
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
							Image: r.getImageName("api", cluster.Spec.DifyVersion),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 5001,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: r.getResourceRequirements(cluster.Spec.Components.API.Resources),
							Env:       r.buildAPIEnvVars(cluster),
						},
					},
				},
			},
		},
	}

	return deployment
}

// buildWorkerDeployment builds the Worker deployment
func (r *DifyClusterReconciler) buildWorkerDeployment(cluster *difyv1alpha1.DifyCluster) *appsv1.Deployment {
	labels := r.buildLabels(cluster.Name, "worker")
	replicas := cluster.Spec.Components.Worker.Replicas
	if replicas == 0 {
		replicas = 3 // 默认值
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-worker", cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, difyv1alpha1.GroupVersion.WithKind("DifyCluster")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
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
							Name:  "worker",
							Image: r.getImageName("worker", cluster.Spec.DifyVersion),
							Command: []string{"python", "-m", "celery", "worker"},
							Resources: r.getResourceRequirements(cluster.Spec.Components.Worker.Resources),
							Env:       r.buildWorkerEnvVars(cluster),
						},
					},
				},
			},
		},
	}

	return deployment
}

// buildWebDeployment builds the Web deployment
func (r *DifyClusterReconciler) buildWebDeployment(cluster *difyv1alpha1.DifyCluster) *appsv1.Deployment {
	labels := r.buildLabels(cluster.Name, "web")
	replicas := cluster.Spec.Components.Web.Replicas
	if replicas == 0 {
		replicas = 2 // 默认值
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-web", cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, difyv1alpha1.GroupVersion.WithKind("DifyCluster")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
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
							Name:  "web",
							Image: r.getImageName("web", cluster.Spec.DifyVersion),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 3000,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: r.getResourceRequirements(cluster.Spec.Components.Web.Resources),
						},
					},
				},
			},
		},
	}

	return deployment
}

// buildSandboxDeployment builds the Sandbox deployment
func (r *DifyClusterReconciler) buildSandboxDeployment(cluster *difyv1alpha1.DifyCluster) *appsv1.Deployment {
	labels := r.buildLabels(cluster.Name, "sandbox")
	replicas := cluster.Spec.Components.Sandbox.Replicas
	if replicas == 0 {
		replicas = 1 // 默认值
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-sandbox", cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, difyv1alpha1.GroupVersion.WithKind("DifyCluster")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
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
							Name:  "sandbox",
							Image: r.getImageName("sandbox", cluster.Spec.DifyVersion),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8194,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: r.getResourceRequirements(cluster.Spec.Components.Sandbox.Resources),
						},
					},
				},
			},
		},
	}

	return deployment
}

// buildAPIService builds the API service
func (r *DifyClusterReconciler) buildAPIService(cluster *difyv1alpha1.DifyCluster) *corev1.Service {
	labels := r.buildLabels(cluster.Name, "api")

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-api", cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, difyv1alpha1.GroupVersion.WithKind("DifyCluster")),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Port:       5001,
					TargetPort: intstr.FromInt(5001),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	return service
}

// buildWebService builds the Web service
func (r *DifyClusterReconciler) buildWebService(cluster *difyv1alpha1.DifyCluster) *corev1.Service {
	labels := r.buildLabels(cluster.Name, "web")

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-web", cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, difyv1alpha1.GroupVersion.WithKind("DifyCluster")),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Port:       3000,
					TargetPort: intstr.FromInt(3000),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	return service
}

// buildSandboxService builds the Sandbox service
func (r *DifyClusterReconciler) buildSandboxService(cluster *difyv1alpha1.DifyCluster) *corev1.Service {
	labels := r.buildLabels(cluster.Name, "sandbox")

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-sandbox", cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, difyv1alpha1.GroupVersion.WithKind("DifyCluster")),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Port:       8194,
					TargetPort: intstr.FromInt(8194),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	return service
}

// buildAPIHPA builds the API HPA
func (r *DifyClusterReconciler) buildAPIHPA(cluster *difyv1alpha1.DifyCluster) *autoscalingv2.HorizontalPodAutoscaler {
	labels := r.buildLabels(cluster.Name, "api")
	autoscaling := cluster.Spec.Components.API.Autoscaling

	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-api", cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, difyv1alpha1.GroupVersion.WithKind("DifyCluster")),
			},
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       fmt.Sprintf("%s-api", cluster.Name),
			},
			MinReplicas: &autoscaling.MinReplicas,
			MaxReplicas: autoscaling.MaxReplicas,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &autoscaling.TargetCPUUtilization,
						},
					},
				},
			},
		},
	}

	return hpa
}

// buildWorkerHPA builds the Worker HPA
func (r *DifyClusterReconciler) buildWorkerHPA(cluster *difyv1alpha1.DifyCluster) *autoscalingv2.HorizontalPodAutoscaler {
	labels := r.buildLabels(cluster.Name, "worker")
	autoscaling := cluster.Spec.Components.Worker.Autoscaling

	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-worker", cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, difyv1alpha1.GroupVersion.WithKind("DifyCluster")),
			},
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       fmt.Sprintf("%s-worker", cluster.Name),
			},
			MinReplicas: &autoscaling.MinReplicas,
			MaxReplicas: autoscaling.MaxReplicas,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &autoscaling.TargetCPUUtilization,
						},
					},
				},
			},
		},
	}

	return hpa
}

// Helper functions

func (r *DifyClusterReconciler) buildLabels(clusterName, component string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      "dify",
		"app.kubernetes.io/component": component,
		"app.kubernetes.io/instance":  clusterName,
	}
}

func (r *DifyClusterReconciler) getImageName(component, version string) string {
	defaultImages := map[string]string{
		"api":     "langgenius/dify-api",
		"worker":  "langgenius/dify-api",
		"web":     "langgenius/dify-web",
		"sandbox": "langgenius/dify-sandbox",
	}

	if version == "" {
		version = "1.7.1"
	}

	// 从环境变量获取默认镜像配置
	if envImage := os.Getenv(fmt.Sprintf("DIFY_DEFAULT_%s_IMAGE", component)); envImage != "" {
		return envImage
	}

	if baseImage, ok := defaultImages[component]; ok {
		return fmt.Sprintf("%s:%s", baseImage, version)
	}

	return fmt.Sprintf("langgenius/dify-%s:%s", component, version)
}

func (r *DifyClusterReconciler) getResourceRequirements(resources corev1.ResourceRequirements) corev1.ResourceRequirements {
	// 如果没有设置资源，使用默认值
	if len(resources.Requests) == 0 && len(resources.Limits) == 0 {
		return corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		}
	}
	return resources
}

func (r *DifyClusterReconciler) buildAPIEnvVars(cluster *difyv1alpha1.DifyCluster) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{Name: "SECRET_KEY", Value: "your-secret-key"},
		{Name: "EDITION", Value: "CLOUD"},
		{Name: "DEPLOY_ENV", Value: "PRODUCTION"},
	}

	// 添加数据库配置
	if cluster.Spec.Database.PostgreSQL != nil {
		pg := cluster.Spec.Database.PostgreSQL
		envVars = append(envVars, corev1.EnvVar{
			Name:  "DATABASE_URL",
			Value: fmt.Sprintf("postgresql://%s:%s@%s:5432/%s", pg.Username, pg.Password, pg.Host, pg.Database),
		})
	}

	if cluster.Spec.Database.Redis != nil {
		redis := cluster.Spec.Database.Redis
		envVars = append(envVars, corev1.EnvVar{
			Name:  "REDIS_URL",
			Value: fmt.Sprintf("redis://:%s@%s:6379/0", redis.Password, redis.Host),
		})
	}

	return envVars
}

func (r *DifyClusterReconciler) buildWorkerEnvVars(cluster *difyv1alpha1.DifyCluster) []corev1.EnvVar {
	// Worker使用与API相同的环境变量
	return r.buildAPIEnvVars(cluster)
}

func (r *DifyClusterReconciler) applyDeployment(ctx context.Context, deployment *appsv1.Deployment) error {
	found := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		return r.Create(ctx, deployment)
	} else if err != nil {
		return err
	}

	// 更新现有的Deployment
	found.Spec = deployment.Spec
	return r.Update(ctx, found)
}

func (r *DifyClusterReconciler) applyService(ctx context.Context, service *corev1.Service) error {
	found := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		return r.Create(ctx, service)
	} else if err != nil {
		return err
	}

	// 更新现有的Service
	found.Spec.Selector = service.Spec.Selector
	found.Spec.Ports = service.Spec.Ports
	return r.Update(ctx, found)
}

func (r *DifyClusterReconciler) applyHPA(ctx context.Context, hpa *autoscalingv2.HorizontalPodAutoscaler) error {
	found := &autoscalingv2.HorizontalPodAutoscaler{}
	err := r.Get(ctx, types.NamespacedName{Name: hpa.Name, Namespace: hpa.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		return r.Create(ctx, hpa)
	} else if err != nil {
		return err
	}

	// 更新现有的HPA
	found.Spec = hpa.Spec
	return r.Update(ctx, found)
}

func (r *DifyClusterReconciler) updateStatus(ctx context.Context, cluster *difyv1alpha1.DifyCluster, phase difyv1alpha1.DifyClusterPhase) error {
	cluster.Status.Phase = phase
	cluster.Status.ObservedGeneration = cluster.Generation
	return r.Status().Update(ctx, cluster)
}

// Utility functions
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

// SetupWithManager sets up the controller with the Manager.
func (r *DifyClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&difyv1alpha1.DifyCluster{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Complete(r)
}