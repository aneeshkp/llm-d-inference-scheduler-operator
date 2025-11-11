# Inference Scheduler Operator

A Kubernetes operator that simplifies deployment and management of distributed LLM inference workloads using the **[llm-d framework](https://github.com/kubernetes-sigs/llm-instance-gateway)**.

## Overview

The Inference Scheduler Operator provides a declarative, Kubernetes-native way to deploy and manage Large Language Model (LLM) inference services. It automates the configuration of intelligent routing, load balancing, and resource management by leveraging the llm-d ecosystem.

**Key Features:**
- **One-CR Deployment** - Single `InferenceScheduler` Custom Resource creates all necessary components
- **Intelligent Routing** - Automatic request routing based on load, prefix cache hits, and GPU utilization
- **Gateway Flexibility** - Support for multiple gateway implementations (kgateway, Istio, GKE)
- **Auto-scaling** - Built-in support for replica management and resource scaling
- **Production Ready** - Follows Kubernetes operator best practices and OLM standards

## Built on llm-d

This operator is built on top of the **llm-d (Large Language Model Distributed) framework**, which provides:
- **Gateway API Inference Extension (GIE)** - Kubernetes-native APIs for AI/ML routing (`InferencePool`, `InferenceService`)
- **Intelligent Endpoint Selection** - Request routing based on cache affinity, load, and GPU metrics
- **Production-Grade Infrastructure** - Battle-tested components from CoreWeave, Google, IBM Research, NVIDIA, and Red Hat

### Why Use This Operator?

| Use llm-d Directly (Helm) | Use Inference Scheduler Operator |
|---------------------------|----------------------------------|
| Multiple Helm chart deployments | Single Kubernetes Custom Resource |
| Manual wiring of components | Automated component orchestration |
| Requires deep llm-d knowledge | Opinionated defaults, easy to start |
| Full control and flexibility | Simplified lifecycle management |
| Best for platform teams | Best for application teams |

**The operator makes llm-d easier to use** - similar to how a Kubernetes Deployment makes Pods easier to manage.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    User                                  │
│  kubectl apply -f inferencescheduler.yaml                │
└────────────────────────┬────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│         Inference Scheduler Operator                     │
│  • Watches InferenceScheduler CRs                        │
│  • Orchestrates llm-d components                         │
│  • Manages lifecycle and updates                         │
└────────────────────────┬────────────────────────────────┘
                         │
                         ├──► Creates: Deployment (vLLM pods)
                         ├──► Creates: Service
                         ├──► Creates: InferencePool (llm-d CRD)
                         └──► Creates: Gateway (Gateway API)

┌─────────────────────────────────────────────────────────┐
│              llm-d Framework (Prerequisites)             │
│  • Gateway API v1.3.0+                                   │
│  • Gateway API Inference Extension (GIE) v1.1.0+         │
│  • Gateway Implementation (kgateway/Istio/GKE)           │
│  • Intelligent request routing and load balancing        │
└─────────────────────────────────────────────────────────┘
```

## Prerequisites

**IMPORTANT:** Before deploying the operator, you must install the llm-d prerequisites:

### Required Components
1. **Gateway API v1.3.0+** - Kubernetes standard for ingress/routing
2. **Gateway API Inference Extension (GIE) v1.1.0+** - AI/ML-specific CRDs
3. **Gateway Implementation** - Choose one:
   - **kgateway** (recommended for AI workloads)
   - **Istio** (for service mesh integration)
   - **GKE Gateway** (for GKE environments)

### Quick Installation

Run the provided installation script:

```bash
cd /path/to/inference-scheduler-operator
./hack/install-prerequisites.sh
```

The script will:
- Verify cluster connectivity
- Install Gateway API CRDs
- Install Gateway API Inference Extension (GIE)
- Help you choose and install a gateway implementation
- Verify all components are ready

### Manual Installation

If you prefer manual installation:

```bash
# Install Gateway API
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml

# Install Gateway API Inference Extension (GIE)
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api-inference-extension/releases/download/v1.1.0/manifests.yaml

# Install kgateway (example)
helm repo add kgateway https://helm.kgateway.io
helm install kgateway kgateway/kgateway --create-namespace --namespace kgateway-system
```

## Getting Started

### 1. Install Prerequisites

```bash
./hack/install-prerequisites.sh
```

### 2. Install the Operator

**Option A: Via kubectl (Development)**

```bash
# Install the CRDs
make install

# Run the operator locally
make run
```

**Option B: Via OLM (Production)**

```bash
operator-sdk run bundle quay.io/aneeshkp/inference-scheduler-operator-bundle:v0.0.1
```

### 3. Create HuggingFace Token Secret

```bash
kubectl create secret generic hf-token \
  --from-literal=token=hf_your_token_here
```

### 4. Deploy an InferenceScheduler

**Minimal GPU Example:**

```yaml
apiVersion: llm.llm-d.io/v1alpha1
kind: InferenceScheduler
metadata:
  name: qwen-inference
spec:
  modelServer:
    type: vllm
    modelName: "Qwen/Qwen2.5-0.5B-Instruct"
    replicas: 2
    hfTokenSecretName: "hf-token"
    resources:
      limits:
        nvidia.com/gpu: "1"
```

Apply it:

```bash
kubectl apply -f config/samples/llm_v1alpha1_inferencescheduler_minimal.yaml
```

**CPU-Only Example (No GPU):**

```bash
kubectl apply -f config/samples/llm_v1alpha1_inferencescheduler_simulator_minimal.yaml
```

**Production with Istio:**

```bash
kubectl apply -f config/samples/llm_v1alpha1_inferencescheduler_istio.yaml
```

### 5. Check Status

```bash
# Watch InferenceScheduler status
kubectl get inferencescheduler -w

# View detailed information
kubectl describe inferencescheduler qwen-inference

# Check created resources
kubectl get all -l app.kubernetes.io/managed-by=inference-scheduler-operator
```

## Sample Configurations

See [config/samples/README.md](config/samples/README.md) for detailed examples:

| Sample | Use Case | GPU | Model | Gateway |
|--------|----------|-----|-------|---------|
| `minimal.yaml` | Quick start | Yes | Qwen 0.5B | kgateway |
| `inferencescheduler.yaml` | Production | Yes | Llama 8B | kgateway |
| `istio.yaml` | Service mesh | Yes | Llama 8B | Istio |
| `simulator_minimal.yaml` | Local testing | No | Qwen 0.5B | kgateway |
| `simulator.yaml` | CPU-only prod | No | Qwen 0.5B | kgateway |

## Configuration

### InferenceScheduler Spec

```yaml
apiVersion: llm.llm-d.io/v1alpha1
kind: InferenceScheduler
metadata:
  name: my-inference
spec:
  # Model Server Configuration
  modelServer:
    type: vllm                                    # Model server type
    modelName: "meta-llama/Llama-3.1-8B-Instruct" # HuggingFace model ID
    replicas: 3                                   # Number of replicas
    enablePrefixCaching: true                     # Enable prefix caching
    gpuMemoryUtilization: 0.9                     # GPU memory utilization
    hfTokenSecretName: "hf-token"                 # HuggingFace token secret
    resources:
      limits:
        nvidia.com/gpu: "1"
        memory: "16Gi"

  # Endpoint Picker Configuration (Intelligent Routing)
  endpointPicker:
    plugins:
      loadAwareScorer:                            # Load-based routing
        enabled: true
        weight: 1.0
      prefixCacheScorer:                          # Cache-aware routing
        enabled: true
        weight: 2.0                               # 2x importance
      kvCacheUtilizationScorer:                   # GPU KV cache routing
        enabled: true
        weight: 1.0

  # Gateway Configuration
  gateway:
    className: "kgateway"                         # kgateway, istio, or gke
    listenerPort: 80
    serviceType: "LoadBalancer"                   # LoadBalancer or ClusterIP
```

## Development

### Prerequisites
- Go v1.24.0+
- Docker v17.03+
- kubectl v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster
- operator-sdk v1.41.1+

### Build and Test

```bash
# Run tests
make test

# Build operator binary
make build

# Build and push container image
make docker-build docker-push IMG=<your-registry>/inference-scheduler-operator:tag

# Install CRDs
make install

# Run locally
make run

# Deploy to cluster
make deploy IMG=<your-registry>/inference-scheduler-operator:tag
```

### Generate Manifests

```bash
# After changing API or controller code
make manifests generate
```

## OLM Bundle

Create an OLM bundle for OperatorHub distribution:

```bash
# Generate bundle
make bundle IMG=quay.io/aneeshkp/inference-scheduler-operator:v0.0.1

# Build and push bundle image
make bundle-build bundle-push BUNDLE_IMG=quay.io/aneeshkp/inference-scheduler-operator-bundle:v0.0.1

# Test via OLM
operator-sdk run bundle quay.io/aneeshkp/inference-scheduler-operator-bundle:v0.0.1
```

## Troubleshooting

### Prerequisites Missing

**Error:** `Prerequisites validation failed: missing prerequisites`

**Solution:** Run the prerequisite installation script:
```bash
./hack/install-prerequisites.sh
```

### Pods Not Starting

**Check events:**
```bash
kubectl describe inferencescheduler <name>
kubectl get events --sort-by='.lastTimestamp'
```

### Model Download Issues

**Verify HuggingFace token:**
```bash
kubectl get secret hf-token -o jsonpath='{.data.token}' | base64 -d
```

### Gateway Not Ready

**Check gateway status:**
```bash
kubectl get gateway -A
kubectl get gatewayclass
```

## Architecture Decisions


See [llm-d documentation](https://github.com/kubernetes-sigs/llm-instance-gateway) for more details on the prerequisites model.

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run `make test` and `make manifests generate`
6. Submit a pull request

## Related Projects

- **[llm-d Framework](https://github.com/kubernetes-sigs/llm-instance-gateway)** - Core distributed LLM inference framework
- **[Gateway API](https://gateway-api.sigs.k8s.io/)** - Kubernetes ingress/routing standard
- **[Gateway API Inference Extension](https://github.com/kubernetes-sigs/gateway-api-inference-extension)** - AI/ML routing extensions
- **[vLLM](https://github.com/vllm-project/vllm)** - High-performance LLM inference engine
- **[kgateway](https://github.com/kgateway-dev/kgateway)** - Gateway API implementation with inference support

## Documentation

- [Sample CRs and Examples](config/samples/README.md)
- [Production Packaging Guide](docs/production-packaging.md)
- [Operator Dependencies Architecture](docs/operator-dependencies.md)

## License

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

---

**Built on the llm-d framework** - Making distributed LLM inference simple and production-ready.
