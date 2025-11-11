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
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	llmv1alpha1 "github.com/aneeshkp/inference-scheduler-operator/api/v1alpha1"
)

const (
	finalizerName = "llm.llm-d.io/finalizer"

	// Default values
	defaultModelServerImage = "vllm/vllm-openai:latest"
	defaultEPPImage        = "ghcr.io/llm-d/llm-d-inference-scheduler:v0.3.2"
	defaultModelServerPort = 8000
	defaultEPPGRPCPort     = 9002
	defaultGatewayPort     = 80
)

// InferenceSchedulerReconciler reconciles a InferenceScheduler object
type InferenceSchedulerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=llm.llm-d.io,resources=inferenceschedulers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=llm.llm-d.io,resources=inferenceschedulers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=llm.llm-d.io,resources=inferenceschedulers/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=inference.networking.k8s.io,resources=inferencepools,verbs=get;list;watch;create;update;patch;delete

func (r *InferenceSchedulerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Starting reconciliation", "name", req.Name, "namespace", req.Namespace)

	// Fetch the InferenceScheduler instance
	infScheduler := &llmv1alpha1.InferenceScheduler{}
	if err := r.Get(ctx, req.NamespacedName, infScheduler); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("InferenceScheduler resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get InferenceScheduler")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !infScheduler.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, infScheduler)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(infScheduler, finalizerName) {
		controllerutil.AddFinalizer(infScheduler, finalizerName)
		if err := r.Update(ctx, infScheduler); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Set initial phase
	if infScheduler.Status.Phase == "" {
		infScheduler.Status.Phase = "Initializing"
		if err := r.Status().Update(ctx, infScheduler); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Phase 1: Validate Prerequisites
	logger.Info("Validating prerequisites (Gateway API, GIE, GatewayClass)")
	if err := r.validatePrerequisites(ctx, infScheduler); err != nil {
		logger.Error(err, "Prerequisites validation failed")
		infScheduler.Status.PrerequisitesValidated = false
		infScheduler.Status.PrerequisiteMessage = err.Error()
		infScheduler.Status.Phase = "PrerequisitesMissing"
		r.updateCondition(infScheduler, "PrerequisitesValidated", metav1.ConditionFalse, "ValidationFailed", err.Error())
		r.Status().Update(ctx, infScheduler)
		// Requeue after 60 seconds to check again
		return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	}

	// Prerequisites validated successfully
	if !infScheduler.Status.PrerequisitesValidated {
		infScheduler.Status.PrerequisitesValidated = true
		infScheduler.Status.PrerequisiteMessage = "All prerequisites validated successfully"
		r.updateCondition(infScheduler, "PrerequisitesValidated", metav1.ConditionTrue, "Validated", "Gateway API, GIE, and GatewayClass are present")
		logger.Info("Prerequisites validated successfully")
	}

	infScheduler.Status.Phase = "Deploying"
	r.Status().Update(ctx, infScheduler)

	// Phase 4: Deploy Model Server
	logger.Info("Deploying model server")

	deployment := r.buildModelServerDeployment(infScheduler)
	if err := r.createOrUpdate(ctx, deployment, infScheduler); err != nil {
		logger.Error(err, "Failed to create/update model server deployment")
		r.updateCondition(infScheduler, "ModelServerReady", metav1.ConditionFalse, "DeploymentFailed", err.Error())
		r.Status().Update(ctx, infScheduler)
		return ctrl.Result{}, err
	}

	service := r.buildModelServerService(infScheduler)
	if err := r.createOrUpdate(ctx, service, infScheduler); err != nil {
		logger.Error(err, "Failed to create/update model server service")
		return ctrl.Result{}, err
	}

	// Check deployment readiness
	ready, err := r.isDeploymentReady(ctx, deployment.Namespace, deployment.Name)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !ready {
		logger.Info("Waiting for model server deployment to be ready")
		r.updateCondition(infScheduler, "ModelServerReady", metav1.ConditionFalse, "NotReady", "Model server pods are not ready yet")
		infScheduler.Status.ModelServerReplicas = 0
		r.Status().Update(ctx, infScheduler)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	r.updateCondition(infScheduler, "ModelServerReady", metav1.ConditionTrue, "Ready", "All model server pods are running")
	infScheduler.Status.ModelServerReplicas = infScheduler.Spec.ModelServer.Replicas

	// Phase 5: Deploy EPP
	logger.Info("Deploying Endpoint Picker (EPP)")

	// Create EPP resources
	sa := r.buildEPPServiceAccount(infScheduler)
	if err := r.createOrUpdate(ctx, sa, infScheduler); err != nil {
		return ctrl.Result{}, err
	}

	role := r.buildEPPRole(infScheduler)
	if err := r.createOrUpdate(ctx, role, infScheduler); err != nil {
		return ctrl.Result{}, err
	}

	roleBinding := r.buildEPPRoleBinding(infScheduler)
	if err := r.createOrUpdate(ctx, roleBinding, infScheduler); err != nil {
		return ctrl.Result{}, err
	}

	configMap := r.buildEPPConfigMap(infScheduler)
	if err := r.createOrUpdate(ctx, configMap, infScheduler); err != nil {
		return ctrl.Result{}, err
	}

	eppDeployment := r.buildEPPDeployment(infScheduler)
	if err := r.createOrUpdate(ctx, eppDeployment, infScheduler); err != nil {
		logger.Error(err, "Failed to create/update EPP deployment")
		r.updateCondition(infScheduler, "EPPReady", metav1.ConditionFalse, "DeploymentFailed", err.Error())
		r.Status().Update(ctx, infScheduler)
		return ctrl.Result{}, err
	}

	eppService := r.buildEPPService(infScheduler)
	if err := r.createOrUpdate(ctx, eppService, infScheduler); err != nil {
		return ctrl.Result{}, err
	}

	// Check EPP readiness
	eppReady, err := r.isDeploymentReady(ctx, eppDeployment.Namespace, eppDeployment.Name)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !eppReady {
		logger.Info("Waiting for EPP deployment to be ready")
		r.updateCondition(infScheduler, "EPPReady", metav1.ConditionFalse, "NotReady", "EPP pods are not ready yet")
		infScheduler.Status.EPPReplicas = 0
		r.Status().Update(ctx, infScheduler)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	r.updateCondition(infScheduler, "EPPReady", metav1.ConditionTrue, "Ready", "EPP is running")
	infScheduler.Status.EPPReplicas = infScheduler.Spec.EndpointPicker.Replicas

	// Phase 6: Create InferencePool
	logger.Info("Creating InferencePool")

	inferencePool := r.buildInferencePool(infScheduler)
	if err := r.createOrUpdateUnstructured(ctx, inferencePool, infScheduler); err != nil {
		logger.Error(err, "Failed to create/update InferencePool")
		r.updateCondition(infScheduler, "InferencePoolReady", metav1.ConditionFalse, "CreationFailed", err.Error())
		r.Status().Update(ctx, infScheduler)
		return ctrl.Result{}, err
	}

	r.updateCondition(infScheduler, "InferencePoolReady", metav1.ConditionTrue, "Ready", "InferencePool created successfully")
	infScheduler.Status.InferencePoolReady = true

	// Phase 7: Create Gateway and HTTPRoute
	logger.Info("Creating Gateway and HTTPRoute")

	gateway := r.buildGateway(infScheduler)
	if err := r.createOrUpdateUnstructured(ctx, gateway, infScheduler); err != nil {
		logger.Error(err, "Failed to create/update Gateway")
		r.updateCondition(infScheduler, "GatewayReady", metav1.ConditionFalse, "CreationFailed", err.Error())
		r.Status().Update(ctx, infScheduler)
		return ctrl.Result{}, err
	}

	httpRoute := r.buildHTTPRoute(infScheduler)
	if err := r.createOrUpdateUnstructured(ctx, httpRoute, infScheduler); err != nil {
		logger.Error(err, "Failed to create/update HTTPRoute")
		return ctrl.Result{}, err
	}

	r.updateCondition(infScheduler, "GatewayReady", metav1.ConditionTrue, "Ready", "Gateway and HTTPRoute created successfully")
	infScheduler.Status.GatewayReady = true

	// Final status update
	infScheduler.Status.Phase = "Ready"
	if err := r.Status().Update(ctx, infScheduler); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Reconciliation complete", "name", infScheduler.Name, "phase", infScheduler.Status.Phase)

	// Requeue after 5 minutes to check health
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// handleDeletion handles the deletion of InferenceScheduler resources
func (r *InferenceSchedulerReconciler) handleDeletion(ctx context.Context, infScheduler *llmv1alpha1.InferenceScheduler) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(infScheduler, finalizerName) {
		return ctrl.Result{}, nil
	}

	logger.Info("Handling deletion", "name", infScheduler.Name)

	// Resources are automatically cleaned up due to owner references
	// Additional cleanup can be added here if needed

	// Remove finalizer
	controllerutil.RemoveFinalizer(infScheduler, finalizerName)
	if err := r.Update(ctx, infScheduler); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Finalizer removed", "name", infScheduler.Name)
	return ctrl.Result{}, nil
}

// validatePrerequisites checks that all required prerequisites are installed
// This follows the llm-d approach: operators declare dependencies, don't install them
func (r *InferenceSchedulerReconciler) validatePrerequisites(ctx context.Context, infScheduler *llmv1alpha1.InferenceScheduler) error {
	var missingPrereqs []string

	// Check Gateway API CRDs exist
	gatewayList := &unstructured.UnstructuredList{}
	gatewayList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "Gateway",
	})
	if err := r.List(ctx, gatewayList, client.Limit(1)); err != nil {
		if meta.IsNoMatchError(err) {
			missingPrereqs = append(missingPrereqs, "Gateway API v1.3.0+ (install: kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml)")
		}
	}

	// Check HTTPRoute CRD exists
	httpRouteList := &unstructured.UnstructuredList{}
	httpRouteList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "HTTPRoute",
	})
	if err := r.List(ctx, httpRouteList, client.Limit(1)); err != nil {
		if meta.IsNoMatchError(err) && !contains(missingPrereqs, "Gateway API") {
			missingPrereqs = append(missingPrereqs, "Gateway API HTTPRoute CRD")
		}
	}

	// Check GIE CRDs exist
	poolList := &unstructured.UnstructuredList{}
	poolList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "inference.networking.k8s.io",
		Version: "v1",
		Kind:    "InferencePool",
	})
	if err := r.List(ctx, poolList, client.Limit(1)); err != nil {
		if meta.IsNoMatchError(err) {
			missingPrereqs = append(missingPrereqs, "Gateway API Inference Extension v1.1.0+ (install: kubectl apply -f https://github.com/kubernetes-sigs/gateway-api-inference-extension/releases/download/v1.1.0/manifests.yaml)")
		}
	}

	// Check GatewayClass exists
	gatewayClassList := &unstructured.UnstructuredList{}
	gatewayClassList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "GatewayClass",
	})
	if err := r.List(ctx, gatewayClassList); err != nil {
		if meta.IsNoMatchError(err) {
			missingPrereqs = append(missingPrereqs, "GatewayClass CRD")
		}
	} else {
		// Check if the requested GatewayClass exists
		gatewayClassName := getDefaultString(infScheduler.Spec.Gateway.ClassName, "kgateway")
		found := false
		for _, item := range gatewayClassList.Items {
			if item.GetName() == gatewayClassName {
				found = true
				break
			}
		}
		if !found {
			missingPrereqs = append(missingPrereqs, fmt.Sprintf("GatewayClass '%s' (install gateway implementation: kgateway, istio, or gke)", gatewayClassName))
		}
	}

	if len(missingPrereqs) > 0 {
		return fmt.Errorf("missing prerequisites: %s. See installation guide: https://github.com/aneeshkp/inference-scheduler-operator/blob/main/README.md#prerequisites", strings.Join(missingPrereqs, "; "))
	}

	return nil
}

// contains checks if a string slice contains a substring
func contains(slice []string, substr string) bool {
	for _, item := range slice {
		if strings.Contains(item, substr) {
			return true
		}
	}
	return false
}

// isDeploymentReady checks if a deployment is ready
func (r *InferenceSchedulerReconciler) isDeploymentReady(ctx context.Context, namespace, name string) (bool, error) {
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, deployment)
	if err != nil {
		return false, err
	}

	// Check if desired replicas match ready replicas
	return deployment.Status.ReadyReplicas == *deployment.Spec.Replicas, nil
}

// createOrUpdate creates or updates a Kubernetes resource
func (r *InferenceSchedulerReconciler) createOrUpdate(ctx context.Context, obj client.Object, owner client.Object) error {
	key := client.ObjectKeyFromObject(obj)
	existing := obj.DeepCopyObject().(client.Object)

	err := r.Get(ctx, key, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			// Set owner reference
			if err := ctrl.SetControllerReference(owner, obj, r.Scheme); err != nil {
				return err
			}
			return r.Create(ctx, obj)
		}
		return err
	}

	// Update existing resource
	obj.SetResourceVersion(existing.GetResourceVersion())
	if err := ctrl.SetControllerReference(owner, obj, r.Scheme); err != nil {
		return err
	}
	return r.Update(ctx, obj)
}

// createOrUpdateUnstructured creates or updates an unstructured resource
func (r *InferenceSchedulerReconciler) createOrUpdateUnstructured(ctx context.Context, obj *unstructured.Unstructured, owner client.Object) error {
	key := client.ObjectKeyFromObject(obj)
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(obj.GroupVersionKind())

	err := r.Get(ctx, key, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			// Set owner reference
			if err := ctrl.SetControllerReference(owner, obj, r.Scheme); err != nil {
				return err
			}
			return r.Create(ctx, obj)
		}
		return err
	}

	// Update existing resource
	obj.SetResourceVersion(existing.GetResourceVersion())
	if err := ctrl.SetControllerReference(owner, obj, r.Scheme); err != nil {
		return err
	}
	return r.Update(ctx, obj)
}

// updateCondition updates or adds a condition to the status
func (r *InferenceSchedulerReconciler) updateCondition(
	infScheduler *llmv1alpha1.InferenceScheduler,
	conditionType string,
	status metav1.ConditionStatus,
	reason, message string,
) {
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: infScheduler.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}

	meta.SetStatusCondition(&infScheduler.Status.Conditions, condition)
}

// sanitizeName sanitizes a string to be a valid Kubernetes name
func sanitizeName(name string) string {
	// Replace invalid characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9\-]`)
	sanitized := reg.ReplaceAllString(strings.ToLower(name), "-")

	// Trim leading/trailing hyphens
	sanitized = strings.Trim(sanitized, "-")

	// Limit length to 63 characters
	if len(sanitized) > 63 {
		sanitized = sanitized[:63]
	}

	return sanitized
}

// getDefaultInt32 returns the value if not nil, otherwise returns default
func getDefaultInt32(value *int32, defaultValue int32) int32 {
	if value != nil {
		return *value
	}
	return defaultValue
}

// getDefaultString returns the value if not empty, otherwise returns default
func getDefaultString(value, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}

// getDefaultFloat64 returns the value if not nil, otherwise returns default
func getDefaultFloat64(value *float64, defaultValue float64) float64 {
	if value != nil {
		return *value
	}
	return defaultValue
}

// SetupWithManager sets up the controller with the Manager.
func (r *InferenceSchedulerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&llmv1alpha1.InferenceScheduler{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Named("inferencescheduler").
		Complete(r)
}
