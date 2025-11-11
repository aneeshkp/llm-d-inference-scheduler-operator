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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InferenceSchedulerSpec defines the desired state of InferenceScheduler
type InferenceSchedulerSpec struct {
	// ModelServer configuration for the inference model (vLLM, TGI, etc.)
	// +kubebuilder:validation:Required
	ModelServer ModelServerSpec `json:"modelServer"`

	// EndpointPicker configuration for intelligent routing
	// +optional
	EndpointPicker EndpointPickerSpec `json:"endpointPicker,omitempty"`

	// Gateway configuration
	// +optional
	Gateway GatewaySpec `json:"gateway,omitempty"`
}

// ModelServerSpec defines the model server configuration
type ModelServerSpec struct {
	// Type of model server (vllm, tgi, etc.)
	// +kubebuilder:validation:Enum=vllm;tgi
	// +kubebuilder:default=vllm
	Type string `json:"type,omitempty"`

	// ModelName is the HuggingFace model name to deploy
	// +kubebuilder:validation:Required
	ModelName string `json:"modelName"`

	// Replicas is the number of model server instances
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=2
	Replicas int32 `json:"replicas,omitempty"`

	// Image is the container image for the model server
	// +kubebuilder:default="vllm/vllm-openai:latest"
	Image string `json:"image,omitempty"`

	// Resources defines resource requirements for model server pods
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// EnablePrefixCaching enables prefix caching in vLLM
	// +kubebuilder:default=true
	EnablePrefixCaching bool `json:"enablePrefixCaching,omitempty"`

	// GPUMemoryUtilization sets the GPU memory utilization (0.0-1.0)
	// +kubebuilder:validation:Minimum=0.0
	// +kubebuilder:validation:Maximum=1.0
	// +kubebuilder:default=0.9
	// +kubebuilder:validation:Type=number
	GPUMemoryUtilization *float64 `json:"gpuMemoryUtilization,omitempty"`

	// HFTokenSecretName is the name of the secret containing HuggingFace token
	// +kubebuilder:validation:Required
	HFTokenSecretName string `json:"hfTokenSecretName"`

	// Port is the HTTP port for the model server
	// +kubebuilder:default=8000
	Port int32 `json:"port,omitempty"`

	// Labels to apply to model server pods
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// EndpointPickerSpec defines the EPP configuration
type EndpointPickerSpec struct {
	// Image is the EPP container image
	// +kubebuilder:default="ghcr.io/llm-d/llm-d-inference-scheduler:v0.3.2"
	Image string `json:"image,omitempty"`

	// Replicas is the number of EPP instances
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas,omitempty"`

	// GRPCPort is the gRPC port for EPP
	// +kubebuilder:default=9002
	GRPCPort int32 `json:"grpcPort,omitempty"`

	// Plugins configuration for routing decisions
	// +optional
	Plugins PluginConfig `json:"plugins,omitempty"`

	// Resources defines resource requirements for EPP pods
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// PluginConfig defines the plugin configuration for EPP
type PluginConfig struct {
	// LoadAwareScorer configuration
	// +optional
	LoadAwareScorer *ScorerPlugin `json:"loadAwareScorer,omitempty"`

	// PrefixCacheScorer configuration
	// +optional
	PrefixCacheScorer *ScorerPlugin `json:"prefixCacheScorer,omitempty"`

	// KVCacheUtilizationScorer configuration
	// +optional
	KVCacheUtilizationScorer *ScorerPlugin `json:"kvCacheUtilizationScorer,omitempty"`
}

// ScorerPlugin defines a scorer plugin configuration
type ScorerPlugin struct {
	// Enabled indicates if this plugin is enabled
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Weight is the weight for this scorer
	// +kubebuilder:default=1.0
	// +kubebuilder:validation:Type=number
	Weight *float64 `json:"weight,omitempty"`

	// Parameters are plugin-specific parameters
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`
}

// GatewaySpec defines the Gateway configuration
type GatewaySpec struct {
	// ClassName is the GatewayClass to use (e.g., "kgateway", "istio", "gke-l7-regional-external-managed")
	// The GatewayClass must be pre-installed in the cluster
	// +kubebuilder:validation:Enum=kgateway;istio;gke-l7-regional-external-managed
	// +kubebuilder:default="kgateway"
	ClassName string `json:"className,omitempty"`

	// ListenerPort is the HTTP listener port
	// +kubebuilder:default=80
	ListenerPort int32 `json:"listenerPort,omitempty"`

	// ServiceType is the Kubernetes Service type (ClusterIP, LoadBalancer, NodePort)
	// +kubebuilder:validation:Enum=ClusterIP;LoadBalancer;NodePort
	// +kubebuilder:default="ClusterIP"
	ServiceType string `json:"serviceType,omitempty"`

	// Name is the name of the Gateway resource to create
	// If not specified, defaults to <InferenceScheduler-name>-gateway
	// +optional
	Name string `json:"name,omitempty"`
}

// InferenceSchedulerStatus defines the observed state of InferenceScheduler
type InferenceSchedulerStatus struct {
	// Conditions represent the latest available observations of the InferenceScheduler's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase indicates the current phase of the deployment
	// +optional
	Phase string `json:"phase,omitempty"`

	// ModelServerReplicas is the current number of model server replicas
	// +optional
	ModelServerReplicas int32 `json:"modelServerReplicas,omitempty"`

	// EPPReplicas is the current number of EPP replicas
	// +optional
	EPPReplicas int32 `json:"eppReplicas,omitempty"`

	// GatewayReady indicates if the Gateway is ready
	// +optional
	GatewayReady bool `json:"gatewayReady,omitempty"`

	// InferencePoolReady indicates if the InferencePool is ready
	// +optional
	InferencePoolReady bool `json:"inferencePoolReady,omitempty"`

	// PrerequisitesValidated indicates if all prerequisites (Gateway API, GIE, GatewayClass) are present
	// +optional
	PrerequisitesValidated bool `json:"prerequisitesValidated,omitempty"`

	// PrerequisiteMessage provides details about missing prerequisites
	// +optional
	PrerequisiteMessage string `json:"prerequisiteMessage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=infsch
// +kubebuilder:printcolumn:name="Model",type=string,JSONPath=`.spec.modelServer.modelName`
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.spec.modelServer.replicas`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// InferenceScheduler is the Schema for the inferenceschedulers API
type InferenceScheduler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InferenceSchedulerSpec   `json:"spec,omitempty"`
	Status InferenceSchedulerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InferenceSchedulerList contains a list of InferenceScheduler
type InferenceSchedulerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InferenceScheduler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InferenceScheduler{}, &InferenceSchedulerList{})
}
