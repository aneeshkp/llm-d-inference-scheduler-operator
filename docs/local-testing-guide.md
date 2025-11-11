# Local Testing Guide - Inference Scheduler Operator

This guide shows how to test the Inference Scheduler Operator locally using either **kind** (recommended) or **minikube**.

## Why kind? (Recommended)

✅ **Faster** - Starts in seconds
✅ **Lighter** - Uses Docker containers, not VMs
✅ **Simpler** - Native Docker integration
✅ **CI/CD friendly** - Easy to automate
✅ **Perfect for operators** - No GPU needed for basic testing

## Prerequisites

### Required Tools

```bash
# Check if you have these installed
docker --version          # Docker 20.10+
kubectl version --client  # kubectl 1.11.3+
kind version             # kind 0.20.0+ (or install below)
```

### Install kind (if needed)

**macOS:**
```bash
brew install kind
```

**Linux:**
```bash
# For AMD64 / x86_64
[ $(uname -m) = x86_64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind
```

**Verify:**
```bash
kind version
```

## Testing Approach - Choose Your Path

### Path 1: Quick Test (Without OLM) ⭐ **START HERE**

**Best for:** Rapid development, debugging, first-time testing

**Pros:**
- ✅ Fastest way to test
- ✅ Easy debugging (operator runs locally with full logs)
- ✅ No OLM complexity
- ✅ Can use IDE debugger

**Cons:**
- ❌ Doesn't test OLM bundle
- ❌ Operator not running in-cluster

---

### Path 2: Full OLM Test (With OLM Bundle)

**Best for:** Testing production deployment, OperatorHub preparation

**Pros:**
- ✅ Tests complete OLM integration
- ✅ Validates bundle and CSV
- ✅ Production-like deployment

**Cons:**
- ❌ Slower iteration
- ❌ Harder to debug
- ❌ Requires OLM installation

---

## Path 1: Quick Test (Without OLM)

### Step 1: Create kind Cluster

```bash
cd /Users/aneeshputtur/github.com/aneeshkp/inference-scheduler-operator

# Create a simple kind cluster
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: inference-test
nodes:
- role: control-plane
  # Expose ports if you want to access services from host
  extraPortMappings:
  - containerPort: 30080
    hostPort: 8080
    protocol: TCP
EOF
```

**Verify cluster:**
```bash
kubectl cluster-info --context kind-inference-test
kubectl get nodes
```

### Step 2: Install Prerequisites

Run the installation script:

```bash
./hack/install-prerequisites.sh
```

**What to select:**
- Gateway implementation: Choose **kgateway** (option 1)
- Wait for all components to be ready

**Verify prerequisites:**
```bash
# Check Gateway API CRDs
kubectl get crd gateways.gateway.networking.k8s.io

# Check GIE CRDs
kubectl get crd inferencepools.inference.networking.k8s.io

# Check GatewayClass
kubectl get gatewayclass
```

### Step 3: Install Operator CRDs

```bash
make install
```

**Verify:**
```bash
kubectl get crd inferenceschedulers.llm.llm-d.io
```

### Step 4: Run Operator Locally

Open a terminal and run:

```bash
make run
```

You should see output like:
```
...
INFO    controller-runtime.metrics      Metrics server is starting to listen
INFO    Starting server
INFO    controller.inferencescheduler   Starting EventSource
INFO    controller.inferencescheduler   Starting Controller
INFO    controller.inferencescheduler   Starting workers
```

**Leave this terminal running!** The operator is now watching for InferenceScheduler CRs.

### Step 5: Create HuggingFace Token Secret

In a **new terminal**:

```bash
# Use your actual HuggingFace token
kubectl create secret generic hf-token \
  --from-literal=token=hf_your_actual_token_here
```

**Don't have a token?** Get one at https://huggingface.co/settings/tokens

### Step 6: Deploy Test InferenceScheduler

**Use the CPU-only sample** (no GPU required):

```bash
kubectl apply -f config/samples/llm_v1alpha1_inferencescheduler_simulator_minimal.yaml
```

### Step 7: Monitor Deployment

**Watch the operator logs** in the first terminal where `make run` is running.

In the second terminal, watch resources:

```bash
# Watch InferenceScheduler status
kubectl get inferencescheduler -w

# In another terminal, watch all resources
kubectl get pods,svc,deployment,inferencepools,gateways -A -w

# View detailed status
kubectl describe inferencescheduler qwen-simulator-minimal

# Check operator created resources
kubectl get all -l app.kubernetes.io/managed-by=inference-scheduler-operator
```

### Step 8: Verify Prerequisites Validation

Check if the operator validated prerequisites correctly:

```bash
kubectl get inferencescheduler qwen-simulator-minimal -o jsonpath='{.status.prerequisitesValidated}'
# Should output: true
```

### Step 9: Test Changes (Iteration Loop)

When you make code changes:

```bash
# 1. Stop the operator (Ctrl+C in the first terminal)

# 2. Regenerate manifests if you changed API
make manifests generate

# 3. Reinstall CRDs if you changed API
make install

# 4. Run again
make run
```

### Step 10: Cleanup

```bash
# Delete InferenceScheduler
kubectl delete inferencescheduler qwen-simulator-minimal

# Delete kind cluster
kind delete cluster --name inference-test
```

---

## Path 2: Full OLM Test (With OLM Bundle)

### Step 1: Create kind Cluster

```bash
cd /Users/aneeshputtur/github.com/aneeshkp/inference-scheduler-operator

# Same as Path 1
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: inference-olm-test
nodes:
- role: control-plane
EOF

kubectl cluster-info --context kind-inference-olm-test
```

### Step 2: Install OLM

```bash
# Install OLM using operator-sdk
operator-sdk olm install
```

**Verify OLM:**
```bash
kubectl get pods -n olm
# Should see: catalog-operator, olm-operator, packageserver pods
```

### Step 3: Install Prerequisites

```bash
./hack/install-prerequisites.sh
```

### Step 4: Build and Load Operator Image

Since kind runs in Docker, you need to build and load the image:

```bash
# Build the operator image
make docker-build IMG=localhost:5000/inference-scheduler-operator:test

# Load image into kind cluster
kind load docker-image localhost:5000/inference-scheduler-operator:test --name inference-olm-test
```

### Step 5: Generate OLM Bundle

```bash
# Generate bundle manifests
make bundle IMG=localhost:5000/inference-scheduler-operator:test

# Review the generated CSV
cat bundle/manifests/inference-scheduler-operator.clusterserviceversion.yaml
```

### Step 6: Build and Load Bundle Image

```bash
# Build bundle image
make bundle-build BUNDLE_IMG=localhost:5000/inference-scheduler-operator-bundle:test

# Load bundle into kind
kind load docker-image localhost:5000/inference-scheduler-operator-bundle:test --name inference-olm-test
```

### Step 7: Run Bundle via OLM

```bash
# Deploy via OLM
operator-sdk run bundle localhost:5000/inference-scheduler-operator-bundle:test
```

**Watch the deployment:**
```bash
kubectl get pods -n operators -w
```

### Step 8: Create HuggingFace Secret and Deploy

```bash
# Create secret
kubectl create secret generic hf-token \
  --from-literal=token=hf_your_token_here

# Deploy InferenceScheduler
kubectl apply -f config/samples/llm_v1alpha1_inferencescheduler_simulator_minimal.yaml
```

### Step 9: Verify OLM Deployment

```bash
# Check CSV status
kubectl get csv -n operators

# Check operator pod logs
kubectl logs -n operators -l control-plane=controller-manager --tail=100 -f

# Check InferenceScheduler
kubectl get inferencescheduler -w
```

### Step 10: Cleanup OLM Test

```bash
# Cleanup operator
operator-sdk cleanup inference-scheduler-operator

# Uninstall OLM (optional)
operator-sdk olm uninstall

# Delete cluster
kind delete cluster --name inference-olm-test
```

---

## Troubleshooting

### Issue: Prerequisites Script Fails

**Error:** `Error: Gateway API installation failed`

**Solution:**
```bash
# Manual installation
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml
kubectl wait --for=condition=Established crd/gateways.gateway.networking.k8s.io --timeout=60s
```

### Issue: Kind Cluster Creation Fails

**Error:** `ERROR: failed to create cluster: node(s) already exist for a cluster with the name "inference-test"`

**Solution:**
```bash
kind delete cluster --name inference-test
kind create cluster --name inference-test
```

### Issue: Image Pull Errors in kind

**Error:** `ImagePullBackOff` when running OLM bundle

**Solution:**
```bash
# Make sure to load the image into kind
kind load docker-image localhost:5000/inference-scheduler-operator:test --name inference-olm-test
kind load docker-image localhost:5000/inference-scheduler-operator-bundle:test --name inference-olm-test

# Verify image is loaded
docker exec -it inference-olm-test-control-plane crictl images | grep inference-scheduler
```

### Issue: Operator Not Finding Prerequisites

**Error:** `Prerequisites validation failed: missing prerequisites`

**Solution:**
```bash
# Verify all prerequisites are installed
kubectl get crd gateways.gateway.networking.k8s.io
kubectl get crd inferencepools.inference.networking.k8s.io
kubectl get gatewayclass

# If missing, run installation script again
./hack/install-prerequisites.sh
```

### Issue: vLLM Pod Pending (CPU-only test)

**Symptom:** Pod stuck in Pending state

**Cause:** Resource requests too high for kind node

**Solution:** Edit the sample to reduce resources:
```yaml
resources:
  requests:
    cpu: "1"      # Reduced from 4
    memory: "2Gi" # Reduced from 8Gi
  limits:
    cpu: "2"
    memory: "4Gi"
```

---

## Using Minikube (Alternative)

If you prefer minikube:

### Setup Minikube

```bash
# Start with sufficient resources
minikube start --cpus=4 --memory=8192 --driver=docker

# Enable registry addon (for local images)
minikube addons enable registry
```

### Follow Same Steps

The testing steps are nearly identical to kind, except:

**For local image loading:**
```bash
# Instead of `kind load docker-image`
eval $(minikube docker-env)
make docker-build IMG=localhost:5000/inference-scheduler-operator:test
```

**For cleanup:**
```bash
minikube delete
```

---

## Comparison: kind vs minikube for This Operator

| Aspect | kind | minikube |
|--------|------|----------|
| **Startup time** | ✅ ~30 seconds | ~2-3 minutes |
| **Resource usage** | ✅ Lower (containers) | Higher (VM) |
| **Docker integration** | ✅ Native | Requires docker-env |
| **Multiple clusters** | ✅ Easy | Harder |
| **Image loading** | `kind load` | `eval $(minikube docker-env)` |
| **Port forwarding** | extraPortMappings | `minikube service` or tunnels |
| **Cleanup** | `kind delete` | `minikube delete` |
| **GPU support** | Limited | ✅ Better (not needed here) |

**Recommendation:** Use **kind** for operator development, especially with CPU-only samples.

---

## Recommended Workflow

### Daily Development (Path 1)

```bash
# 1. Start cluster (once per day)
kind create cluster --name inference-test

# 2. Install prerequisites (once per cluster)
./hack/install-prerequisites.sh

# 3. Development loop (many times)
make install              # After API changes
make run                  # Run operator
# ... test, make changes ...
Ctrl+C                    # Stop operator
make manifests generate   # After changes
make run                  # Run again

# 4. End of day
kind delete cluster --name inference-test
```

### Pre-Release Testing (Path 2)

```bash
# Test OLM bundle before publishing
kind create cluster --name inference-olm-test
operator-sdk olm install
./hack/install-prerequisites.sh
make docker-build IMG=test-image:latest
kind load docker-image test-image:latest
make bundle IMG=test-image:latest
make bundle-build bundle-push BUNDLE_IMG=test-bundle:latest
operator-sdk run bundle test-bundle:latest

# Verify everything works
kubectl apply -f config/samples/llm_v1alpha1_inferencescheduler_simulator_minimal.yaml
kubectl get inferencescheduler -w

# Cleanup
operator-sdk cleanup inference-scheduler-operator
kind delete cluster --name inference-olm-test
```

---

## Next Steps

After successful local testing:

1. **Run Tests:** `make test`
2. **Build for Production:** `make docker-build docker-push IMG=quay.io/aneeshkp/inference-scheduler-operator:v0.0.1`
3. **Create Bundle:** `make bundle`
4. **Test on Real Cluster:** Deploy to a development Kubernetes cluster
5. **Publish to OperatorHub:** Submit bundle to community-operators repository

---

## Quick Reference

### kind Commands
```bash
kind create cluster --name NAME
kind get clusters
kind delete cluster --name NAME
kind load docker-image IMAGE --name CLUSTER
```

### Operator Development
```bash
make install              # Install CRDs
make run                  # Run operator locally
make manifests generate   # Regenerate manifests
make docker-build IMG=... # Build image
make bundle IMG=...       # Generate OLM bundle
```

### Debugging
```bash
kubectl get inferencescheduler NAME -o yaml
kubectl describe inferencescheduler NAME
kubectl logs -n NAMESPACE pod/POD-NAME -f
kubectl get events --sort-by='.lastTimestamp'
```

---

**Recommended:** Start with **Path 1 (Quick Test)** using **kind**. It's the fastest way to verify your operator works!
