# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

The K8s Cluster Agent is a lightweight, read-only Kubernetes service that runs inside a cluster and provides a RESTful API for querying pod and node information. It follows clean architecture principles with clear separation between transport, business logic, and Kubernetes client layers.

## Common Development Commands

### Building and Running
```bash
make build              # Build local binary
make docker-build       # Build Docker image
make run               # Run locally (requires kubeconfig)
```

### Testing and Quality
```bash
make test              # Run unit tests with coverage
make test-integration  # Run integration tests
make lint             # Run golangci-lint
make coverage         # View HTML coverage report
make check            # Run all checks (tidy, lint, test)
```

### Deployment
```bash
make deploy-dev        # Deploy to development (debug logging, single replica)
make deploy-prod       # Deploy to production (with HPA, network policies)
```

### Development Utilities
```bash
make generate-mocks    # Generate mocks from interfaces
make deps             # Install development dependencies
make tidy             # Clean up go.mod dependencies
```

## Architecture

### Core Structure
- **cmd/agent/**: Entry point with graceful shutdown handling
- **internal/core/**: Business logic layer - services, models, interfaces
- **internal/transport/http/**: REST API - handlers, middleware, router
- **internal/kubernetes/**: K8s client initialization
- **deployments/**: Kustomize-based K8s manifests (base + overlays)

### Key Design Principles
1. **Dependency Direction**: Transport → Core → Kubernetes (never reverse)
2. **Interface Segregation**: Small, focused interfaces for each service
3. **Error Handling**: Wrap errors with context, use typed errors
4. **Security First**: Read-only access, minimal RBAC, non-root container

### API Endpoints
- `GET /api/v1/pods/{namespace}/{podName}/describe` - Full pod description
- `GET /api/v1/pods/{namespace}/{podName}/scheduling` - Enhanced scheduling info with failure analysis
- `GET /api/v1/pods/{namespace}/{podName}/resources` - Resource requirements
- `GET /api/v1/pods/{namespace}/{podName}/failure-events` - Analyzed failure events with insights
- `GET /api/v1/nodes/{nodeName}/utilization` - Node metrics (requires metrics server)

#### Enhanced Scheduling API
The `/scheduling` endpoint provides comprehensive scheduling analysis:

**For Scheduled Pods:**
- Reasons why the pod was placed on its current node
- Matched constraints (affinity, tolerations, resources)
- Scheduling event history

**For Pending Pods:**
- Per-node analysis of scheduling failures
- Categorized failure reasons:
  - Resource constraints: `InsufficientCPU`, `InsufficientMemory`, `InsufficientStorage`
  - Volume issues: `VolumeAttachmentError`, `VolumeMultiAttachError`, `VolumeNodeAffinityConflict`
  - Scheduling constraints: `NodeAffinityNotMatch`, `TaintTolerationMismatch`, `PodAffinityConflict`
  - Node status: `NodeNotReady`
  - Other: `Miscellaneous`
- Failure summary with affected node counts
- Parsed scheduling events (FailedScheduling, etc.)

Example response structure for pending pod:
```json
{
  "status": "Pending",
  "failureCategories": ["InsufficientMemory", "VolumeNodeAffinityConflict"],
  "failureSummary": [
    {
      "category": "InsufficientMemory",
      "count": 3,
      "description": "Insufficient memory resources available on nodes",
      "nodes": ["node-1", "node-2", "node-3"]
    }
  ],
  "unschedulableNodes": [
    {
      "nodeName": "node-1",
      "reasons": ["insufficient memory", "PV zone doesn't match node"],
      "insufficientResources": ["insufficient memory (requested: 2Gi, allocatable: 1Gi)"]
    }
  ]
}
```

#### Failure Events API
The `/failure-events` endpoint provides intelligent analysis of pod failure events:

**Features:**
- Categorizes failures into meaningful groups (Scheduling, ImagePull, ContainerCrash, Volume, Resource, Probe, Network)
- Provides severity classification (critical, warning, info)
- Tracks recurrence patterns and calculates rates
- Offers actionable insights with possible causes and suggested actions
- Identifies ongoing issues (last 5 minutes)

Example response structure:
```json
{
  "podName": "my-pod",
  "namespace": "default",
  "totalEvents": 15,
  "failureEvents": [
    {
      "type": "Warning",
      "reason": "CrashLoopBackOff",
      "message": "Back-off restarting failed container",
      "category": "ContainerCrash",
      "severity": "critical",
      "isRecurring": true,
      "recurrenceRate": "6.0 times per hour",
      "timeSinceFirst": "1h25m",
      "possibleCauses": ["Application crash", "Missing dependencies"],
      "suggestedAction": "Examine container logs and fix application startup issues"
    }
  ],
  "eventCategories": {"ContainerCrash": 1, "Scheduling": 1},
  "criticalEvents": 2,
  "warningEvents": 0,
  "ongoingIssues": ["CrashLoopBackOff: Back-off restarting failed container"]
}
```

### Request Flow
1. HTTP handler validates request and extracts parameters
2. Handler calls core service with context (includes timeout)
3. Service uses K8s client to fetch data
4. Response formatted with consistent structure (data + metadata)
5. Middleware logs request with duration and request ID

### Testing Strategy
- Unit tests use fake K8s clients (not mocks for K8s APIs)
- Table-driven tests for comprehensive coverage
- Integration tests with build tags
- Minimum 80% coverage, 100% for core business logic

### Configuration
Environment variables control behavior:
- `PORT` (default: 8080)
- `LOG_LEVEL` (default: info) 
- `LOG_FORMAT` (default: json)
- `K8S_TIMEOUT` (default: 30s)
- Request timeout fixed at 500ms for high performance

### Security Considerations
- ServiceAccount with minimal RBAC (get/list pods, get nodes/metrics)
- Container runs as non-root user (65534)
- Read-only root filesystem
- No privilege escalation, all capabilities dropped

## Important Project Rules

From .cursor/rules/rules.mdc:
- Always use structured logging with slog
- 500ms timeout on all API requests
- Return typed errors with proper HTTP status codes
- No business logic in HTTP handlers
- Use context for cancellation and request tracing
- Generate mocks with mockgen for testing
- Follow standard Go conventions and gofmt