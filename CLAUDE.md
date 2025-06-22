# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

The K8s Cluster Agent is a lightweight, read-only Kubernetes service that runs inside a cluster and provides a RESTful API for querying pod and node information. It follows clean architecture principles with clear separation between transport, business logic, and Kubernetes client layers.

## Common Development Commands
- Don't add any comments

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
- `GET /api/v1/pods/{namespace}/{podName}/scheduling/explain` - Detailed scheduling explanation (like Elasticsearch's allocation explain)
- `GET /api/v1/pods/{namespace}/{podName}/health-score` - Pod health score with detailed component analysis
- `GET /api/v1/nodes/{nodeName}/utilization` - Node metrics (requires metrics server)
- `GET /api/v1/cluster/pod-issues` - Cluster-wide pod issues dashboard with pattern detection

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

#### Scheduling Explain API
The `/scheduling/explain` endpoint provides Elasticsearch-style detailed explanations for pod scheduling decisions:

**Features:**
- Per-node detailed analysis showing exactly why a pod can/cannot be scheduled
- Resource calculations with exact numbers:
  - Node capacity, allocatable, currently allocated, and available resources
  - Exact shortage amounts when pod doesn't fit
  - Percentage utilization for each resource type
- Detailed affinity/anti-affinity explanations:
  - Which labels are missing for node selectors
  - Which node affinity terms failed and why
  - Pod affinity/anti-affinity conflicts with specific pods
- Taint/toleration analysis with specific recommendations
- Volume constraint checks including PV node affinity
- Actionable recommendations for resolving scheduling issues

Example response structure:
```json
{
  "podName": "my-pod",
  "namespace": "default",
  "status": "Pending",
  "nodeAnalysis": [
    {
      "nodeName": "node-1",
      "schedulable": false,
      "reasons": {
        "resources": {
          "fits": false,
          "details": {
            "cpu": {
              "podRequests": "2000m",
              "nodeCapacity": "8000m",
              "nodeAllocatable": "7820m",
              "nodeAllocated": "6500m",
              "nodeAvailable": "1320m",
              "shortage": "680m",
              "percentUsed": 83.12,
              "recommendation": "Pod needs 680m more cpu than available on this node"
            }
          }
        },
        "affinity": {
          "nodeSelector": {
            "matched": false,
            "required": {"zone": "us-west-1"},
            "missingLabels": ["zone=us-west-1"],
            "details": "Node selector requirements not met. Missing labels: zone=us-west-1"
          }
        }
      },
      "recommendation": "Node cannot schedule pod due to: needs 680m cpu, node selector mismatch"
    }
  ],
  "summary": {
    "totalNodes": 10,
    "filteredByResources": 8,
    "filteredByNodeSelector": 2,
    "recommendation": "No nodes have sufficient resources. Consider scaling up the cluster or reducing pod resource requests.",
    "possibleActions": [
      "Reduce pod cpu request by at least 680m",
      "Scale up cluster by adding more nodes",
      "Enable cluster autoscaler if not already enabled"
    ]
  }
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

#### Pod Health Score API
The `/health-score` endpoint provides a comprehensive health assessment for pods:

**Features:**
- Calculates overall health score (0-100) based on multiple factors
- Component-based scoring with weighted contributions:
  - Container restarts (30% weight)
  - Container states (25% weight)
  - Recent events (20% weight)
  - Pod conditions (15% weight)
  - Uptime/stability (10% weight)
- Provides detailed health metrics including restart frequency and uptime
- Categorizes health status: Healthy, Good, Warning, Degraded, Critical

Example response structure:
```json
{
  "podName": "my-pod",
  "namespace": "default",
  "overallScore": 75,
  "status": "Good",
  "components": {
    "restarts": {
      "name": "Container Restarts",
      "score": 70,
      "weight": 0.30,
      "status": "Good",
      "description": "5 total restarts"
    },
    "containerStates": {
      "name": "Container States",
      "score": 100,
      "weight": 0.25,
      "status": "Excellent",
      "description": "2/2 containers healthy"
    }
  },
  "details": {
    "restartCount": 5,
    "restartFrequency": "0.42 restarts/hour",
    "uptime": "12h 30m",
    "containerStatuses": [...],
    "recentEvents": [...]
  }
}
```

#### Cluster-Wide Pod Issues Dashboard API
The `/cluster/pod-issues` endpoint provides a real-time aggregated view of pod problems:

**Features:**
- Aggregates pod issues across all namespaces or filtered by namespace
- Categorizes issues: CrashLoopBackOff, ImagePullError, Pending, OOMKilled, etc.
- Tracks issue velocity and trends (improving/stable/degrading)
- Detects patterns across multiple pods
- Provides namespace-level breakdown
- Supports severity filtering

Example response structure:
```json
{
  "totalPods": 150,
  "healthyPods": 120,
  "unhealthyPods": 30,
  "issueCategories": {
    "CrashLoopBackOff": 5,
    "ImagePullError": 3,
    "PendingScheduling": 2
  },
  "topIssues": [
    {
      "category": "CrashLoopBackOff",
      "count": 5,
      "severity": "critical",
      "description": "Container repeatedly crashing and restarting",
      "affectedPods": ["default/app-1", "default/app-2", "..."]
    }
  ],
  "issueVelocity": {
    "newIssuesLastHour": 3,
    "trendDirection": "degrading",
    "velocityPerHour": 2.5
  },
  "patterns": [
    {
      "type": "ImagePullError",
      "description": "ImagePullError: ErrImagePull",
      "count": 3,
      "namespaces": ["default", "production"],
      "commonLabels": {"app": "frontend"}
    }
  ]
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