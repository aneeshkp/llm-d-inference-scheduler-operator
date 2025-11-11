# Development Cluster Management

This document describes the Makefile targets for creating and managing local development clusters for the Inference Scheduler Operator.

## Quick Reference

### KIND Cluster Management

```bash
# Create KIND cluster
make kind-create

# Delete KIND cluster
make kind-delete

# Install OLM on KIND
make kind-olm-install

# Create cluster + Install OLM (all-in-one)
make kind-setup
```

### Minikube Cluster Management

```bash
# Create minikube cluster
make minikube-create

# Delete minikube cluster
make minikube-delete

# Install OLM on minikube
make minikube-olm-install

# Create cluster + Install OLM (all-in-one)
make minikube-setup
```

## Configuration

You can customize cluster names and OLM version using environment variables:

```bash
# Custom KIND cluster name
KIND_CLUSTER_NAME=my-cluster make kind-create

# Custom minikube profile
MINIKUBE_PROFILE=my-profile make minikube-create

# Custom OLM version
OLM_VERSION=v0.28.0 make kind-olm-install
```

### Default Values

| Variable | Default Value |
|----------|---------------|
| `KIND_CLUSTER_NAME` | `inference-scheduler-dev` |
| `MINIKUBE_PROFILE` | `inference-scheduler-dev` |
| `OLM_VERSION` | `v0.30.0` |

## Complete Development Workflow

### Using KIND

```bash
# 1. Create cluster with OLM
make kind-setup

# 2. Install prerequisites (Gateway API, GIE, kgateway)
./hack/install-prerequisites.sh

# 3. Create HuggingFace token secret
kubectl create secret generic hf-token --from-literal=token=hf_your_token

# 4. Deploy operator (option A: run locally)
make install  # Install CRDs
make run      # Run operator locally

# 4. Deploy operator (option B: deploy to cluster)
export IMG=quay.io/aneeshkp/inference-scheduler-operator:dev
make docker-build docker-push IMG=$IMG
make deploy IMG=$IMG

# 5. Create InferenceScheduler
kubectl apply -f config/samples/llm_v1alpha1_inferencescheduler_minimal.yaml

# 6. Monitor deployment
kubectl get inferencescheduler -w
kubectl describe inferencescheduler

# 7. Cleanup
make kind-delete
```

### Using Minikube

```bash
# 1. Create cluster with OLM
make minikube-setup

# 2. Install prerequisites
./hack/install-prerequisites.sh

# 3. Create HuggingFace token secret
kubectl create secret generic hf-token --from-literal=token=hf_your_token

# 4. Deploy operator
export IMG=quay.io/aneeshkp/inference-scheduler-operator:dev
make docker-build docker-push IMG=$IMG
make deploy IMG=$IMG

# 5. Create InferenceScheduler
kubectl apply -f config/samples/llm_v1alpha1_inferencescheduler_minimal.yaml

# 6. Monitor deployment
kubectl get inferencescheduler -w

# 7. Cleanup
make minikube-delete
```

## What Each Target Does

### `make kind-create`

Creates a KIND (Kubernetes in Docker) cluster with the name specified by `KIND_CLUSTER_NAME`.

- Checks if KIND is installed
- Creates cluster if it doesn't exist
- Skips creation if cluster already exists

### `make kind-delete`

Deletes the KIND cluster.

- Checks if cluster exists
- Deletes cluster and all resources

### `make kind-olm-install`

Installs Operator Lifecycle Manager (OLM) on the KIND cluster.

- Switches to the KIND cluster context
- Downloads and installs OLM from official releases
- Waits for OLM components to be ready
- Skips if OLM is already installed

### `make kind-setup`

All-in-one target that:
1. Creates KIND cluster (`make kind-create`)
2. Installs OLM (`make kind-olm-install`)
3. Displays next steps

**This is the recommended way to set up a development environment.**

### `make minikube-create`

Creates a minikube cluster with the profile name specified by `MINIKUBE_PROFILE`.

- Checks if minikube is installed
- Creates cluster with 4 CPUs and 8GB memory
- Skips creation if profile already exists

### `make minikube-delete`

Deletes the minikube cluster.

### `make minikube-olm-install`

Installs OLM on the minikube cluster.

### `make minikube-setup`

All-in-one target for minikube (same as KIND setup).

## Prerequisites

Before using these targets, ensure you have the required tools installed:

### For KIND

```bash
# Install KIND
# macOS
brew install kind

# Linux
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind
```

### For Minikube

```bash
# Install minikube
# macOS
brew install minikube

# Linux
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube
```

### Common Tools

```bash
# kubectl (required for both)
# macOS
brew install kubectl

# Linux
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/
```

## Troubleshooting

### KIND cluster already exists

```bash
# List existing clusters
kind get clusters

# Delete if needed
make kind-delete
# or
kind delete cluster --name inference-scheduler-dev
```

### minikube profile already exists

```bash
# List profiles
minikube profile list

# Delete if needed
make minikube-delete
# or
minikube delete -p inference-scheduler-dev
```

### OLM installation fails

```bash
# Check OLM pods
kubectl get pods -n olm

# Check OLM deployments
kubectl get deployments -n olm

# View logs
kubectl logs -n olm deployment/olm-operator
kubectl logs -n olm deployment/catalog-operator
```

### Context not switching

```bash
# Manually switch context to KIND
kubectl config use-context kind-inference-scheduler-dev

# Manually switch context to minikube
kubectl config use-context inference-scheduler-dev
```

## Examples

### Create multiple test environments

```bash
# Environment 1: KIND
KIND_CLUSTER_NAME=test-env-1 make kind-setup

# Environment 2: minikube
MINIKUBE_PROFILE=test-env-2 make minikube-setup

# List all contexts
kubectl config get-contexts

# Switch between them
kubectl config use-context kind-test-env-1
kubectl config use-context test-env-2
```

### Test with specific OLM version

```bash
# Use older OLM version
OLM_VERSION=v0.28.0 make kind-setup
```

### Quick teardown and rebuild

```bash
# Complete reset
make kind-delete
make kind-setup
./hack/install-prerequisites.sh
```

## Integration with CI/CD

These targets can be used in CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
name: Test Operator
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'

      - name: Install KIND
        run: |
          curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
          chmod +x ./kind
          sudo mv ./kind /usr/local/bin/kind

      - name: Create test cluster
        run: make kind-setup

      - name: Install prerequisites
        run: ./hack/install-prerequisites.sh

      - name: Run tests
        run: make test-e2e

      - name: Cleanup
        if: always()
        run: make kind-delete
```

## See Also

- [Operator Development Guide](../README.md)
- [Prerequisites Installation](../hack/install-prerequisites.sh)
- [Sample Custom Resources](../config/samples/)
