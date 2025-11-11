# Inference Scheduler Operator - Documentation

This directory contains detailed documentation for developing and deploying the Inference Scheduler Operator.

## Quick Links

- **[Development Clusters Guide](DEVELOPMENT_CLUSTERS.md)** ⭐ **START HERE** - Creating KIND/minikube clusters with OLM
- **[Local Testing Guide](local-testing-guide.md)** - Testing the operator locally
- **[Main README](../README.md)** - Project overview and getting started

## Quick Start

### 1. Create Development Cluster

```bash
# Using KIND (recommended)
make kind-setup

# Or using minikube
make minikube-setup
```

### 2. Install Prerequisites

```bash
./hack/install-prerequisites.sh
```

### 3. Create HuggingFace Secret

```bash
kubectl create secret generic hf-token --from-literal=token=hf_your_token
```

### 4. Run Operator

```bash
# Option A: Run locally (for development)
make install
make run

# Option B: Deploy to cluster
export IMG=quay.io/aneeshkp/inference-scheduler-operator:dev
make docker-build docker-push IMG=$IMG
make deploy IMG=$IMG
```

### 5. Create InferenceScheduler

```bash
kubectl apply -f config/samples/llm_v1alpha1_inferencescheduler_minimal.yaml
```

### 6. Monitor

```bash
kubectl get inferencescheduler -w
kubectl describe inferencescheduler
```

### 7. Cleanup

```bash
make kind-delete
# or
make minikube-delete
```

## Documentation Index

### Getting Started
- [Development Clusters](DEVELOPMENT_CLUSTERS.md) - Creating local Kubernetes clusters
- [Local Testing Guide](local-testing-guide.md) - Testing operator functionality

### Development Workflow
1. **Create Cluster:** `make kind-setup` or `make minikube-setup`
2. **Install Prerequisites:** `./hack/install-prerequisites.sh`
3. **Deploy Operator:** `make install && make run`
4. **Test Changes:** Create sample InferenceSchedulers
5. **Cleanup:** `make kind-delete`

### Available Makefile Targets

#### Development Clusters
```bash
make kind-create          # Create KIND cluster
make kind-delete          # Delete KIND cluster
make kind-olm-install     # Install OLM on KIND
make kind-setup           # Create KIND + OLM (all-in-one)

make minikube-create      # Create minikube cluster
make minikube-delete      # Delete minikube cluster
make minikube-olm-install # Install OLM on minikube
make minikube-setup       # Create minikube + OLM (all-in-one)
```

#### Operator Development
```bash
make manifests            # Generate CRDs and RBAC
make generate             # Generate DeepCopy code
make build                # Build manager binary
make run                  # Run locally
make test                 # Run tests
```

#### Container Images
```bash
make docker-build         # Build operator image
make docker-push          # Push operator image
make docker-buildx        # Multi-arch build
```

#### Deployment
```bash
make install              # Install CRDs
make uninstall            # Uninstall CRDs
make deploy               # Deploy to cluster
make undeploy             # Remove from cluster
```

#### OLM Bundles
```bash
make bundle               # Generate OLM bundle
make bundle-build         # Build bundle image
make bundle-push          # Push bundle image
make catalog-build        # Build catalog image
make catalog-push         # Push catalog image
```

## Prerequisites

### Required Tools

- **Go 1.22+** - For building the operator
- **kubectl** - For cluster interaction
- **operator-sdk v1.41.1** - For operator development
- **Docker or Podman** - For building images

### Optional Tools (for clusters)

- **KIND** - For local Kubernetes clusters
- **minikube** - Alternative local cluster
- **Helm 3** - For prerequisite installation

### Installation

```bash
# macOS
brew install go kubectl kind minikube operator-sdk

# Linux
# See individual tool documentation
```

## Configuration

### Cluster Configuration

```bash
# Override cluster names
KIND_CLUSTER_NAME=my-cluster make kind-setup
MINIKUBE_PROFILE=my-profile make minikube-setup

# Override OLM version
OLM_VERSION=v0.28.0 make kind-olm-install
```

### Operator Configuration

```bash
# Override image
IMG=quay.io/myuser/operator:tag make docker-build

# Override version
VERSION=0.0.2 make bundle
```

## Sample Custom Resources

Located in `config/samples/`:

| File | Description |
|------|-------------|
| `llm_v1alpha1_inferencescheduler.yaml` | Full example with all options |
| `llm_v1alpha1_inferencescheduler_minimal.yaml` | Minimal GPU example |
| `llm_v1alpha1_inferencescheduler_istio.yaml` | Using Istio gateway |
| `llm_v1alpha1_inferencescheduler_simulator.yaml` | Full CPU example |
| `llm_v1alpha1_inferencescheduler_simulator_minimal.yaml` | Minimal CPU example |

## Common Workflows

### Testing a New Feature

```bash
# 1. Create fresh cluster
make kind-setup

# 2. Install prerequisites
./hack/install-prerequisites.sh

# 3. Make your code changes
# ... edit files ...

# 4. Test locally
make install
make run

# 5. Create test CR
kubectl apply -f config/samples/llm_v1alpha1_inferencescheduler_minimal.yaml

# 6. Verify
kubectl get inferencescheduler
kubectl logs -f <operator-pod>

# 7. Cleanup
make kind-delete
```

### Building and Publishing

```bash
# 1. Build multi-arch image
export IMG=quay.io/aneeshkp/inference-scheduler-operator:v0.0.1
make docker-buildx IMG=$IMG

# 2. Generate OLM bundle
make bundle IMG=$IMG

# 3. Build and push bundle
export BUNDLE_IMG=quay.io/aneeshkp/inference-scheduler-operator-bundle:v0.0.1
make bundle-build bundle-push BUNDLE_IMG=$BUNDLE_IMG

# 4. Test bundle
operator-sdk run bundle $BUNDLE_IMG
```

### Debugging

```bash
# View operator logs (local mode)
make run

# View operator logs (deployed)
kubectl logs -n inference-scheduler-operator-system deployment/inference-scheduler-operator-controller-manager

# Check CRD installation
kubectl get crd inferenceschedulers.llm.llm-d.io

# Describe InferenceScheduler
kubectl describe inferencescheduler <name>

# Check events
kubectl get events --sort-by='.lastTimestamp'
```

## Troubleshooting

### Cluster Issues

```bash
# KIND cluster not starting
make kind-delete
make kind-create

# minikube not starting
make minikube-delete
make minikube-create

# OLM not ready
kubectl get pods -n olm
kubectl logs -n olm deployment/olm-operator
```

### Operator Issues

```bash
# CRDs not installing
make manifests
make install

# Operator not reconciling
kubectl logs -f deployment/inference-scheduler-operator-controller-manager

# Check RBAC
kubectl auth can-i create deployments --as=system:serviceaccount:default:inference-scheduler-operator-controller-manager
```

## External Resources

- [Operator SDK Documentation](https://sdk.operatorframework.io/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [KIND Documentation](https://kind.sigs.k8s.io/)
- [OLM Documentation](https://olm.operatorframework.io/)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly using local clusters
5. Submit a pull request

## Support

For issues and questions:
- GitHub Issues: (create repository first)
- Documentation: This directory
- Obsidian Vault: `/Users/aneeshputtur/Documents/Obsidian Vault/llm-d/Inference Scheduler/`

---

**Last Updated:** November 10, 2025
