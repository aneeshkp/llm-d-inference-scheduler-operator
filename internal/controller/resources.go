/*
Copyright 2025 Aneesh Puttur.

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

package controller

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"

	llmv1alpha1 "github.com/aneeshkp/inference-scheduler-operator/api/v1alpha1"
)

// buildModelServerDeployment creates a Deployment for the model server (vLLM)
func (r *InferenceSchedulerReconciler) buildModelServerDeployment(infScheduler *llmv1alpha1.InferenceScheduler) *appsv1.Deployment {
	modelName := sanitizeName(infScheduler.Spec.ModelServer.ModelName)

	labels := map[string]string{
		"app":                         "vllm",
		"model":                       modelName,
		"app.kubernetes.io/name":      "model-server",
		"app.kubernetes.io/instance":  infScheduler.Name,
		"app.kubernetes.io/component": "inference",
	}

	// Merge user-provided labels
	for k, v := range infScheduler.Spec.ModelServer.Labels {
		labels[k] = v
	}

	replicas := getDefaultInt32(&infScheduler.Spec.ModelServer.Replicas, 2)
	image := getDefaultString(infScheduler.Spec.ModelServer.Image, defaultModelServerImage)
	port := getDefaultInt32(&infScheduler.Spec.ModelServer.Port, defaultModelServerPort)

	// Build container args
	args := []string{
		fmt.Sprintf("--model=%s", infScheduler.Spec.ModelServer.ModelName),
		fmt.Sprintf("--port=%d", port),
	}

	if infScheduler.Spec.ModelServer.EnablePrefixCaching {
		args = append(args, "--enable-prefix-caching")
	}

	gpuUtil := getDefaultFloat64(infScheduler.Spec.ModelServer.GPUMemoryUtilization, 0.9)
	args = append(args, fmt.Sprintf("--gpu-memory-utilization=%.2f", gpuUtil))

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-vllm", infScheduler.Name),
			Namespace: infScheduler.Namespace,
			Labels:    labels,
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
							Name:  "vllm",
							Image: image,
							Args:  args,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: port,
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: infScheduler.Spec.ModelServer.Resources,
							Env: []corev1.EnvVar{
								{
									Name: "HF_TOKEN",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: infScheduler.Spec.ModelServer.HFTokenSecretName,
											},
											Key: "token",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return deployment
}

// buildModelServerService creates a Service for the model server
func (r *InferenceSchedulerReconciler) buildModelServerService(infScheduler *llmv1alpha1.InferenceScheduler) *corev1.Service {
	modelName := sanitizeName(infScheduler.Spec.ModelServer.ModelName)

	labels := map[string]string{
		"app":   "vllm",
		"model": modelName,
	}

	port := getDefaultInt32(&infScheduler.Spec.ModelServer.Port, defaultModelServerPort)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-vllm", infScheduler.Name),
			Namespace: infScheduler.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       port,
					TargetPort: intstr.FromInt(int(port)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	return service
}

// buildEPPServiceAccount creates a ServiceAccount for EPP
func (r *InferenceSchedulerReconciler) buildEPPServiceAccount(infScheduler *llmv1alpha1.InferenceScheduler) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-epp", infScheduler.Name),
			Namespace: infScheduler.Namespace,
		},
	}
}

// buildEPPRole creates a Role for EPP with permissions to list pods and get inferencepools
func (r *InferenceSchedulerReconciler) buildEPPRole(infScheduler *llmv1alpha1.InferenceScheduler) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-epp", infScheduler.Name),
			Namespace: infScheduler.Namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"inference.networking.k8s.io"},
				Resources: []string{"inferencepools"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
}

// buildEPPRoleBinding creates a RoleBinding for EPP
func (r *InferenceSchedulerReconciler) buildEPPRoleBinding(infScheduler *llmv1alpha1.InferenceScheduler) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-epp", infScheduler.Name),
			Namespace: infScheduler.Namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      fmt.Sprintf("%s-epp", infScheduler.Name),
				Namespace: infScheduler.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     fmt.Sprintf("%s-epp", infScheduler.Name),
		},
	}
}

// buildEPPConfigMap creates a ConfigMap with EPP plugin configuration
func (r *InferenceSchedulerReconciler) buildEPPConfigMap(infScheduler *llmv1alpha1.InferenceScheduler) *corev1.ConfigMap {
	// Build plugin configuration YAML
	pluginConfig := `apiVersion: inference.networking.x-k8s.io/v1alpha1
kind: EndpointPickerConfig
plugins:`

	// Load-aware scorer
	if infScheduler.Spec.EndpointPicker.Plugins.LoadAwareScorer != nil && infScheduler.Spec.EndpointPicker.Plugins.LoadAwareScorer.Enabled {
		weight := getDefaultFloat64(infScheduler.Spec.EndpointPicker.Plugins.LoadAwareScorer.Weight, 1.0)
		pluginConfig += fmt.Sprintf(`
  - type: load-aware-scorer
    weight: %.1f
    parameters:
      queueThreshold: "%s"`,
			weight,
			getDefaultString(infScheduler.Spec.EndpointPicker.Plugins.LoadAwareScorer.Parameters["queueThreshold"], "128"))
	}

	// Prefix cache scorer
	if infScheduler.Spec.EndpointPicker.Plugins.PrefixCacheScorer != nil && infScheduler.Spec.EndpointPicker.Plugins.PrefixCacheScorer.Enabled {
		weight := getDefaultFloat64(infScheduler.Spec.EndpointPicker.Plugins.PrefixCacheScorer.Weight, 2.0)
		pluginConfig += fmt.Sprintf(`
  - type: prefix-cache-scorer
    weight: %.1f
    parameters:
      cacheHitBonus: "%s"`,
			weight,
			getDefaultString(infScheduler.Spec.EndpointPicker.Plugins.PrefixCacheScorer.Parameters["cacheHitBonus"], "1.0"))
	}

	// KV cache utilization scorer
	if infScheduler.Spec.EndpointPicker.Plugins.KVCacheUtilizationScorer != nil && infScheduler.Spec.EndpointPicker.Plugins.KVCacheUtilizationScorer.Enabled {
		weight := getDefaultFloat64(infScheduler.Spec.EndpointPicker.Plugins.KVCacheUtilizationScorer.Weight, 1.0)
		pluginConfig += fmt.Sprintf(`
  - type: kv-cache-utilization-scorer
    weight: %.1f`,
			weight)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-epp-config", infScheduler.Name),
			Namespace: infScheduler.Namespace,
		},
		Data: map[string]string{
			"plugins.yaml": pluginConfig,
		},
	}
}

// buildEPPDeployment creates a Deployment for EPP
func (r *InferenceSchedulerReconciler) buildEPPDeployment(infScheduler *llmv1alpha1.InferenceScheduler) *appsv1.Deployment {
	labels := map[string]string{
		"app":                         "epp",
		"app.kubernetes.io/name":      "endpoint-picker",
		"app.kubernetes.io/instance":  infScheduler.Name,
		"app.kubernetes.io/component": "routing",
	}

	replicas := getDefaultInt32(&infScheduler.Spec.EndpointPicker.Replicas, 1)
	image := getDefaultString(infScheduler.Spec.EndpointPicker.Image, defaultEPPImage)
	grpcPort := getDefaultInt32(&infScheduler.Spec.EndpointPicker.GRPCPort, defaultEPPGRPCPort)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-epp", infScheduler.Name),
			Namespace: infScheduler.Namespace,
			Labels:    labels,
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
					ServiceAccountName: fmt.Sprintf("%s-epp", infScheduler.Name),
					Containers: []corev1.Container{
						{
							Name:  "epp",
							Image: image,
							Args: []string{
								fmt.Sprintf("--pool-name=%s-pool", infScheduler.Name),
								fmt.Sprintf("--pool-namespace=%s", infScheduler.Namespace),
								fmt.Sprintf("--grpc-port=%d", grpcPort),
								"--grpc-health-port=9003",
								"--config-file=/config/plugins.yaml",
								"--v=2",
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: grpcPort,
									Name:          "grpc",
									Protocol:      corev1.ProtocolTCP,
								},
								{
									ContainerPort: 9003,
									Name:          "health",
									Protocol:      corev1.ProtocolTCP,
								},
								{
									ContainerPort: 9090,
									Name:          "metrics",
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: infScheduler.Spec.EndpointPicker.Resources,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/config",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-epp-config", infScheduler.Name),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return deployment
}

// buildEPPService creates a Service for EPP (gRPC)
func (r *InferenceSchedulerReconciler) buildEPPService(infScheduler *llmv1alpha1.InferenceScheduler) *corev1.Service {
	labels := map[string]string{
		"app": "epp",
	}

	grpcPort := getDefaultInt32(&infScheduler.Spec.EndpointPicker.GRPCPort, defaultEPPGRPCPort)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-epp", infScheduler.Name),
			Namespace: infScheduler.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "grpc",
					Port:       grpcPort,
					TargetPort: intstr.FromInt(int(grpcPort)),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "health",
					Port:       9003,
					TargetPort: intstr.FromInt(9003),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "metrics",
					Port:       9090,
					TargetPort: intstr.FromInt(9090),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	return service
}

// buildInferencePool creates an InferencePool CR
func (r *InferenceSchedulerReconciler) buildInferencePool(infScheduler *llmv1alpha1.InferenceScheduler) *unstructured.Unstructured {
	modelName := sanitizeName(infScheduler.Spec.ModelServer.ModelName)

	labels := map[string]string{
		"app":   "vllm",
		"model": modelName,
	}

	grpcPort := getDefaultInt32(&infScheduler.Spec.EndpointPicker.GRPCPort, defaultEPPGRPCPort)
	modelServerPort := getDefaultInt32(&infScheduler.Spec.ModelServer.Port, defaultModelServerPort)

	pool := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "inference.networking.k8s.io/v1",
			"kind":       "InferencePool",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("%s-pool", infScheduler.Name),
				"namespace": infScheduler.Namespace,
			},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{
					"matchLabels": labels,
				},
				"targetPorts": []interface{}{
					map[string]interface{}{
						"number": modelServerPort,
					},
				},
				"endpointPickerRef": map[string]interface{}{
					"name":        fmt.Sprintf("%s-epp", infScheduler.Name),
					"port":        grpcPort,
					"failureMode": "FailOpen",
				},
			},
		},
	}

	return pool
}

// buildGateway creates a Gateway resource
func (r *InferenceSchedulerReconciler) buildGateway(infScheduler *llmv1alpha1.InferenceScheduler) *unstructured.Unstructured {
	className := getDefaultString(infScheduler.Spec.Gateway.ClassName, "kgateway")
	listenerPort := getDefaultInt32(&infScheduler.Spec.Gateway.ListenerPort, defaultGatewayPort)

	gateway := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.networking.k8s.io/v1",
			"kind":       "Gateway",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("%s-gateway", infScheduler.Name),
				"namespace": infScheduler.Namespace,
			},
			"spec": map[string]interface{}{
				"gatewayClassName": className,
				"listeners": []interface{}{
					map[string]interface{}{
						"name":     "http",
						"protocol": "HTTP",
						"port":     listenerPort,
						"allowedRoutes": map[string]interface{}{
							"namespaces": map[string]interface{}{
								"from": "Same",
							},
						},
					},
				},
			},
		},
	}

	return gateway
}

// buildHTTPRoute creates an HTTPRoute resource
func (r *InferenceSchedulerReconciler) buildHTTPRoute(infScheduler *llmv1alpha1.InferenceScheduler) *unstructured.Unstructured {
	modelServerPort := getDefaultInt32(&infScheduler.Spec.ModelServer.Port, defaultModelServerPort)

	httpRoute := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.networking.k8s.io/v1",
			"kind":       "HTTPRoute",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("%s-route", infScheduler.Name),
				"namespace": infScheduler.Namespace,
			},
			"spec": map[string]interface{}{
				"parentRefs": []interface{}{
					map[string]interface{}{
						"name":      fmt.Sprintf("%s-gateway", infScheduler.Name),
						"namespace": infScheduler.Namespace,
					},
				},
				"rules": []interface{}{
					map[string]interface{}{
						"matches": []interface{}{
							map[string]interface{}{
								"path": map[string]interface{}{
									"type":  "PathPrefix",
									"value": "/v1/",
								},
							},
						},
						"backendRefs": []interface{}{
							map[string]interface{}{
								"group": "inference.networking.k8s.io",
								"kind":  "InferencePool",
								"name":  fmt.Sprintf("%s-pool", infScheduler.Name),
								"port":  modelServerPort,
							},
						},
					},
				},
			},
		},
	}

	return httpRoute
}
