#!/bin/bash

# Script to validate Kubernetes manifests

set -e

echo "Validating Kubernetes manifests..."

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo "kubectl is not installed. Please install kubectl to validate manifests."
    exit 1
fi

# Validate base manifests
echo "Validating base manifests..."
for file in deployments/base/*.yaml; do
    if [[ -f "$file" ]]; then
        echo "  Validating $file"
        kubectl --dry-run=client apply -f "$file" > /dev/null
    fi
done

# Validate development overlay
echo "Validating development overlay..."
kubectl --dry-run=client apply -k deployments/overlays/development > /dev/null

# Validate production overlay
echo "Validating production overlay..."
kubectl --dry-run=client apply -k deployments/overlays/production > /dev/null

echo "All manifests are valid!" 