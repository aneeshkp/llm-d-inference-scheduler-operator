#!/bin/bash
# Prerequisite installation script for Inference Scheduler Operator
# Based on llm-d approach: https://github.com/llm-d-incubation/llm-d-infra

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}"
echo "═══════════════════════════════════════════════════════════════"
echo "  Inference Scheduler Operator - Prerequisites Installation"
echo "═══════════════════════════════════════════════════════════════"
echo -e "${NC}"
echo ""
echo "This script installs the required prerequisites:"
echo "  1. Gateway API v1.3.0+"
echo "  2. Gateway API Inference Extension (GIE) v1.1.0+"
echo "  3. Gateway implementation (kgateway, Istio, or skip)"
echo ""

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to print success message
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

# Function to print error message
print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to print warning message
print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# Function to print info message
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"
if ! command_exists kubectl; then
    print_error "kubectl not found. Please install kubectl first."
    echo "  Installation: https://kubernetes.io/docs/tasks/tools/install-kubectl/"
    exit 1
fi
print_success "kubectl found: $(kubectl version --client --short 2>/dev/null || kubectl version --client)"

if ! command_exists helm; then
    print_error "helm not found. Please install Helm first."
    echo "  Installation: https://helm.sh/docs/intro/install/"
    exit 1
fi
print_success "helm found: $(helm version --short)"

# Check cluster connection
if ! kubectl cluster-info >/dev/null 2>&1; then
    print_error "Cannot connect to Kubernetes cluster"
    echo "  Please ensure your kubeconfig is correctly configured"
    exit 1
fi
print_success "Connected to Kubernetes cluster"
echo ""

# Step 1: Install Gateway API
echo -e "${BLUE}[1/4] Installing Gateway API v1.3.0...${NC}"
if kubectl get crd gateways.gateway.networking.k8s.io >/dev/null 2>&1; then
    print_warning "Gateway API CRDs already installed"
    kubectl get crd gateways.gateway.networking.k8s.io -o jsonpath='{.metadata.labels.gateway\.networking\.k8s\.io/bundle-version}' | xargs -I {} echo "  Current version: {}"
else
    kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml
    kubectl wait --for condition=established --timeout=60s \
        crd/gateways.gateway.networking.k8s.io \
        crd/httproutes.gateway.networking.k8s.io \
        crd/gatewayclasses.gateway.networking.k8s.io >/dev/null 2>&1
    print_success "Gateway API CRDs installed"
fi
echo ""

# Step 2: Install Gateway API Inference Extension
echo -e "${BLUE}[2/4] Installing Gateway API Inference Extension v1.1.0...${NC}"
if kubectl get crd inferencepools.inference.networking.k8s.io >/dev/null 2>&1; then
    print_warning "GIE CRDs already installed"
else
    kubectl apply -f https://github.com/kubernetes-sigs/gateway-api-inference-extension/releases/download/v1.1.0/manifests.yaml
    kubectl wait --for condition=established --timeout=60s \
        crd/inferencepools.inference.networking.k8s.io >/dev/null 2>&1
    print_success "GIE CRDs installed"
fi
echo ""

# Step 3: Choose Gateway Implementation
echo -e "${BLUE}[3/4] Choose Gateway Implementation${NC}"
echo ""
echo "Select which Gateway implementation to install:"
echo "  1) kgateway (recommended for AI workloads)"
echo "  2) Istio (for service mesh integration)"
echo "  3) Skip (already installed or will install manually)"
echo ""
read -p "Enter choice [1-3]: " gateway_choice
echo ""

case $gateway_choice in
    1)
        echo -e "${BLUE}Installing kgateway v2.0.2...${NC}"

        # Check if already installed
        if kubectl get deployment kgateway -n kgateway-system >/dev/null 2>&1; then
            print_warning "kgateway already installed"
        else
            # Install kgateway CRDs
            helm upgrade -i --create-namespace --namespace kgateway-system \
                --version v2.0.2 kgateway-crds \
                oci://cr.kgateway.dev/kgateway-dev/charts/kgateway-crds

            # Install kgateway
            helm upgrade -i --namespace kgateway-system \
                --version v2.0.2 --set inferenceExtension.enabled=true \
                kgateway oci://cr.kgateway.dev/kgateway-dev/charts/kgateway

            # Wait for kgateway to be ready
            echo "  Waiting for kgateway to be ready..."
            kubectl wait --for=condition=available --timeout=300s \
                deployment/kgateway -n kgateway-system

            print_success "kgateway installed and ready"
        fi
        ;;
    2)
        echo -e "${BLUE}Installing Istio...${NC}"

        if ! command_exists istioctl; then
            print_error "istioctl not found"
            echo "  Please install istioctl: https://istio.io/latest/docs/setup/install/"
            exit 1
        fi

        # Check if Istio is already installed
        if kubectl get deployment istiod -n istio-system >/dev/null 2>&1; then
            print_warning "Istio already installed"
        else
            istioctl install -y
            print_success "Istio installed"
        fi
        ;;
    3)
        print_info "Skipping gateway installation"
        print_warning "Make sure you have a Gateway implementation installed before using the operator"
        ;;
    *)
        print_error "Invalid choice"
        exit 1
        ;;
esac
echo ""

# Step 4: Verify Installation
echo -e "${BLUE}[4/4] Verifying installation...${NC}"
echo ""

# Check Gateway API CRDs
echo "Gateway API CRDs:"
kubectl get crd | grep gateway.networking.k8s.io | awk '{print "  ✓ " $1}'
echo ""

# Check GIE CRDs
echo "GIE CRDs:"
kubectl get crd | grep inference.networking.k8s.io | awk '{print "  ✓ " $1}'
echo ""

# Check GatewayClasses
echo "Available GatewayClasses:"
if kubectl get gatewayclass >/dev/null 2>&1; then
    kubectl get gatewayclass -o custom-columns=NAME:.metadata.name,CONTROLLER:.spec.controllerName --no-headers | awk '{print "  ✓ " $1 " (controller: " $2 ")"}'
else
    print_warning "No GatewayClasses found"
    echo "  You must have a Gateway implementation installed"
fi
echo ""

# Final success message
print_success "All prerequisites installed successfully!"
echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}  Next Steps:${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo "1. Install the Inference Scheduler Operator:"
echo -e "   ${BLUE}operator-sdk run bundle quay.io/aneeshkp/inference-scheduler-operator-bundle:v0.0.1${NC}"
echo ""
echo "   Or via kubectl:"
echo -e "   ${BLUE}kubectl apply -f https://operatorhub.io/install/inference-scheduler-operator.yaml${NC}"
echo ""
echo "2. Create a HuggingFace token secret:"
echo -e "   ${BLUE}kubectl create secret generic hf-token --from-literal=token=hf_xxx${NC}"
echo ""
echo "3. Deploy an InferenceScheduler:"
echo -e "   ${BLUE}kubectl apply -f config/samples/llm_v1alpha1_inferencescheduler.yaml${NC}"
echo ""
echo "For more information, see the README:"
echo "  https://github.com/aneeshkp/inference-scheduler-operator/blob/main/README.md"
echo ""
