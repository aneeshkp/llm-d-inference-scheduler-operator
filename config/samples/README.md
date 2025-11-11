# InferenceScheduler Sample Custom Resources

This directory contains sample Custom Resources (CRs) demonstrating various configurations for the InferenceScheduler operator.

## Prerequisites

**IMPORTANT:** Before deploying any sample, you must install the required prerequisites:

```bash
# Run the prerequisite installation script
./hack/install-prerequisites.sh
```

This installs:
- Gateway API v1.3.0+
- Gateway API Inference Extension (GIE) v1.1.0+
- Gateway implementation (kgateway, Istio, or your choice)

## Available Samples

### 1. Minimal GPU Example
**File:** `llm_v1alpha1_inferencescheduler_minimal.yaml`

Minimal configuration with GPU resources, suitable for quick testing.

```bash
kubectl apply -f llm_v1alpha1_inferencescheduler_minimal.yaml
```

**Features:**
- Single GPU per pod
- 2 replicas
- Qwen 2.5-0.5B model
- Default EPP and Gateway configuration

---

### 2. Full Production Example
**File:** `llm_v1alpha1_inferencescheduler.yaml`

Complete configuration with all options, suitable for production deployment.

```bash
kubectl apply -f llm_v1alpha1_inferencescheduler.yaml
```

**Features:**
- Auto-install Gateway API, GIE, and kgateway
- Llama 3.1-8B model
- 3 replicas with GPU
- All EPP plugins enabled with custom weights
- LoadBalancer service type

---

### 3. CPU-Only Minimal Example
**File:** `llm_v1alpha1_inferencescheduler_simulator_minimal.yaml`

Minimal CPU-only configuration, ideal for local testing without GPU resources.

```bash
kubectl apply -f llm_v1alpha1_inferencescheduler_simulator_minimal.yaml
```

**Features:**
- CPU-only resources (no GPU)
- 2 replicas
- Qwen 2.5-0.5B model
- Load-aware scoring only
- ClusterIP service type

**Use Cases:**
- Local development
- CI/CD testing
- Environments without GPU

---

### 4. Istio Gateway Example
**File:** `llm_v1alpha1_inferencescheduler_istio.yaml`

Production configuration using Istio as the Gateway implementation.

```bash
kubectl apply -f llm_v1alpha1_inferencescheduler_istio.yaml
```

**Features:**
- Istio gateway (service mesh integration)
- Llama 3.1-8B model
- 3 replicas with GPU
- All EPP plugins enabled
- LoadBalancer service type

**Use Cases:**
- Istio-based service mesh environments
- Multi-cluster deployments
- Advanced traffic management needs

---

### 5. CPU-Only Full Example
**File:** `llm_v1alpha1_inferencescheduler_simulator.yaml`

Complete CPU-only configuration with all options.

```bash
kubectl apply -f llm_v1alpha1_inferencescheduler_simulator.yaml
```

**Features:**
- Auto-install dependencies
- CPU-only resources (4-8 CPU, 8-16Gi memory)
- 2 replicas
- Load-aware scoring enabled
- GPU-specific plugins disabled

**Use Cases:**
- Testing complete operator functionality without GPU
- Development environments
- Simulator/mock deployments

---

## Quick Start

### Prerequisites

1. Create HuggingFace token secret:
```bash
kubectl create secret generic hf-token --from-literal=token=hf_your_token_here
```

2. Install the operator:
```bash
cd /Users/aneeshputtur/github.com/aneeshkp/inference-scheduler-operator
make install
make run
```

### Deploy a Sample

Choose the appropriate sample based on your environment:

**For GPU environments:**
```bash
kubectl apply -f config/samples/llm_v1alpha1_inferencescheduler_minimal.yaml
```

**For CPU-only environments:**
```bash
kubectl apply -f config/samples/llm_v1alpha1_inferencescheduler_simulator_minimal.yaml
```

### Check Status

```bash
# Watch InferenceScheduler status
kubectl get inferencescheduler -w

# Check all created resources
kubectl get all -l app.kubernetes.io/managed-by=inference-scheduler-operator

# View detailed status
kubectl describe inferencescheduler <name>
```

## Sample Comparison

| Feature | Minimal GPU | Full GPU | Istio | CPU Minimal | CPU Full |
|---------|------------|----------|-------|-------------|----------|
| GPU Required | Yes | Yes | Yes | No | No |
| Gateway | kgateway | kgateway | istio | kgateway | kgateway |
| Replicas | 2 | 3 | 3 | 2 | 2 |
| Model | Qwen 0.5B | Llama 8B | Llama 8B | Qwen 0.5B | Qwen 0.5B |
| EPP Plugins | Default | All | All | Load-aware only | Load-aware only |
| Service Type | Default | LoadBalancer | LoadBalancer | ClusterIP | ClusterIP |
| Prefix Caching | Enabled | Enabled | Enabled | Disabled | Disabled |

## Customization

You can customize any sample by modifying the spec. Common customizations:

### Change Model
```yaml
spec:
  modelServer:
    modelName: "meta-llama/Llama-3.1-8B-Instruct"
```

### Adjust Replicas
```yaml
spec:
  modelServer:
    replicas: 5
```

### Change Service Type
```yaml
spec:
  gateway:
    serviceType: "LoadBalancer"  # or ClusterIP, NodePort
```

### Enable/Disable Auto-Install
```yaml
spec:
  installGatewayAPI: false
  installGIE: false
  installKgateway: false
```

### Adjust EPP Plugin Weights
```yaml
spec:
  endpointPicker:
    plugins:
      loadAwareScorer:
        enabled: true
        weight: 1.0
      prefixCacheScorer:
        enabled: true
        weight: 3.0  # 3x importance for cache hits
```

## Troubleshooting

### Pods Not Starting

Check events:
```bash
kubectl describe inferencescheduler <name>
kubectl get events --sort-by='.lastTimestamp'
```

### GPU Not Available

If you see GPU-related errors in CPU-only environments, use the CPU-only samples instead.

### Model Download Issues

Ensure your HuggingFace token is valid:
```bash
kubectl get secret hf-token -o jsonpath='{.data.token}' | base64 -d
```

## Documentation

For complete documentation, see:
- [Inference Scheduler Operator Guide](/Users/aneeshputtur/Documents/Obsidian Vault/llm-d/Inference Scheduler/Inference Scheduler Operator Guide.md)
- [llm-d Inference Scheduler Guide](/Users/aneeshputtur/Documents/Obsidian Vault/llm-d/Inference Scheduler/llm-d Inference Scheduler - Complete Learning & Deployment Guide.md)
