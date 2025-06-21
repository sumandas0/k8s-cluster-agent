# K8s Cluster Agent

A lightweight in-cluster Kubernetes agent that provides read-only access to pod and node information through a RESTful API.

## Features

- **Pod Information**: Get detailed pod information including scheduling details and resource requirements
- **Enhanced Scheduling Analysis**: Comprehensive scheduling failure analysis for pending pods with per-node breakdown
- **Failure Event Analysis**: Intelligent analysis of pod failure events with categorization and actionable insights
- **Namespace Error Analysis**: Analyze all pods in a namespace for common issues (restarts, pending, crashes)
- **Node Utilization**: Retrieve real-time node resource utilization metrics
- **Secure by Default**: Minimal RBAC permissions, read-only access
- **High Performance**: 500ms request timeout, efficient resource usage
- **Production Ready**: Health checks, structured logging, graceful shutdown
- **Flexible Deployment**: Support for both Deployment and DaemonSet modes

## Architecture

The agent runs inside a Kubernetes cluster and exposes a REST API for querying cluster information. It uses the Kubernetes API and metrics server to gather data.

### Components

- **Core Services**: Business logic for interacting with Kubernetes API
- **HTTP Transport**: RESTful API endpoints with middleware for logging, timeout, and recovery
- **Kubernetes Client**: In-cluster client for accessing Kubernetes API and metrics

## Installation

### Prerequisites

- Kubernetes cluster (v1.19+)
- Metrics server installed (for node utilization endpoints)
- kubectl configured to access your cluster

### Quick Start

1. Clone the repository:
```bash
git clone https://github.com/sumandas0/k8s-cluster-agent.git
cd k8s-cluster-agent
```

2. Build the Docker image:
```bash
make docker-build
```

3. Deploy to Kubernetes:

**Development (single replica with debug logging):**
```bash
make deploy-dev
```

**Production (with HPA and network policies):**
```bash
make deploy-prod
```

### Deployment Options

The agent supports multiple deployment strategies:

#### Deployment (Default)
Standard deployment with configurable replicas:
```bash
kubectl apply -k deployments/overlays/development
```

#### DaemonSet
For running one instance per node:
```bash
kubectl apply -f deployments/base/deployment.yaml
```

Configure the deployment type in your overlay by setting the appropriate patches.

## API Documentation

### Base URL
```
http://<service-name>.<namespace>.svc.cluster.local/api/v1
```

### Endpoints

#### Get Pod Description
```http
GET /api/v1/pods/{namespace}/{podName}/describe
```

Returns comprehensive pod information similar to `kubectl describe pod`, including containers, volumes, events, and status details.

**Example:**
```bash
curl http://k8s-cluster-agent.k8s-cluster-agent.svc.cluster.local/api/v1/pods/default/my-pod/describe
```

**Response:**
```json
{
  "data": {
    "name": "my-pod",
    "namespace": "default",
    "labels": {
      "app": "my-app",
      "version": "1.0"
    },
    "annotations": {
      "deployment.kubernetes.io/revision": "1"
    },
    "status": {
      "phase": "Running",
      "podIP": "10.244.1.5",
      "hostIP": "192.168.1.100"
    },
    "node": "node-1",
    "containers": [
      {
        "name": "app",
        "image": "nginx:1.20",
        "ready": true,
        "restartCount": 0,
        "state": {
          "running": {
            "startedAt": "2023-06-21T10:25:00Z"
          }
        },
        "resources": {
          "requests": {
            "cpu": "100m",
            "memory": "128Mi"
          },
          "limits": {
            "cpu": "200m",
            "memory": "256Mi"
          }
        },
        "environment": [
          {
            "name": "ENV_VAR",
            "value": "value"
          }
        ],
        "mounts": [
          {
            "name": "config-volume",
            "mountPath": "/etc/config",
            "readOnly": true
          }
        ]
      }
    ],
    "volumes": [
      {
        "name": "config-volume",
        "type": "ConfigMap",
        "source": {
          "configMap": {
            "name": "my-config"
          }
        }
      }
    ],
    "qosClass": "Burstable",
    "priority": 0,
    "tolerations": [],
    "nodeSelector": {},
    "conditions": [
      {
        "type": "Ready",
        "status": "True",
        "lastTransitionTime": "2023-06-21T10:25:30Z"
      }
    ],
    "events": [
      {
        "type": "Normal",
        "reason": "Scheduled",
        "message": "Successfully assigned pod to node",
        "firstTimestamp": "2023-06-21T10:25:00Z",
        "lastTimestamp": "2023-06-21T10:25:00Z",
        "count": 1,
        "source": "default-scheduler/master-node"
      }
    ]
  },
  "metadata": {
    "requestId": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2023-06-21T10:30:00Z"
  }
}
```

#### Get Pod Scheduling Information
```http
GET /api/v1/pods/{namespace}/{podName}/scheduling
```

Returns enhanced scheduling information for a pod, including comprehensive failure analysis for pending pods.

**Example:**
```bash
curl http://k8s-cluster-agent.k8s-cluster-agent.svc.cluster.local/api/v1/pods/default/my-pod/scheduling
```

**Response for Scheduled Pod:**
```json
{
  "data": {
    "status": "Scheduled",
    "nodeName": "node-1",
    "schedulerName": "default-scheduler",
    "nodeSelector": {
      "kubernetes.io/os": "linux"
    },
    "tolerations": [],
    "affinity": null,
    "priority": 0,
    "priorityClassName": "",
    "schedulingReasons": [
      "Node node-1 matched pod affinity requirements",
      "Node has sufficient resources (CPU: 2/4 cores, Memory: 4Gi/8Gi)"
    ],
    "schedulingEvents": [
      {
        "type": "Normal",
        "reason": "Scheduled",
        "message": "Successfully assigned default/my-pod to node-1",
        "timestamp": "2023-06-21T10:25:00Z"
      }
    ]
  },
  "metadata": {
    "requestId": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2023-06-21T10:30:00Z"
  }
}
```

**Response for Pending Pod with Scheduling Failures:**
```json
{
  "data": {
    "status": "Pending",
    "failureCategories": ["InsufficientMemory", "VolumeNodeAffinityConflict"],
    "failureSummary": [
      {
        "category": "InsufficientMemory",
        "count": 3,
        "description": "Insufficient memory resources available on nodes",
        "nodes": ["node-1", "node-2", "node-3"]
      },
      {
        "category": "VolumeNodeAffinityConflict",
        "count": 1,
        "description": "Persistent volume zone affinity doesn't match node zone",
        "nodes": ["node-4"]
      }
    ],
    "unschedulableNodes": [
      {
        "nodeName": "node-1",
        "reasons": ["insufficient memory", "PV zone doesn't match node"],
        "insufficientResources": ["insufficient memory (requested: 2Gi, allocatable: 1Gi)"],
        "categories": ["InsufficientMemory", "VolumeNodeAffinityConflict"]
      }
    ],
    "schedulingEvents": [
      {
        "type": "Warning",
        "reason": "FailedScheduling",
        "message": "0/4 nodes are available: 3 insufficient memory, 1 node(s) had volume affinity conflict",
        "timestamp": "2023-06-21T10:25:00Z"
      }
    ]
  },
  "metadata": {
    "requestId": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2023-06-21T10:30:00Z"
  }
}
```

**Failure Categories:**
- `InsufficientCPU`: Not enough CPU resources
- `InsufficientMemory`: Not enough memory resources
- `InsufficientStorage`: Not enough ephemeral storage
- `VolumeAttachmentError`: Volume attachment issues
- `VolumeMultiAttachError`: Volume attached to multiple nodes
- `VolumeNodeAffinityConflict`: Volume zone doesn't match node
- `NodeAffinityNotMatch`: Pod node affinity requirements not met
- `TaintTolerationMismatch`: Node taints not tolerated by pod
- `PodAffinityConflict`: Pod affinity/anti-affinity conflicts
- `NodeNotReady`: Node is not in ready state
- `Miscellaneous`: Other scheduling failures

#### Get Pod Resources
```http
GET /api/v1/pods/{namespace}/{podName}/resources
```

Returns aggregated resource requirements for all containers in a pod.

**Example:**
```bash
curl http://k8s-cluster-agent.k8s-cluster-agent.svc.cluster.local/api/v1/pods/default/my-pod/resources
```

**Response:**
```json
{
  "data": {
    "containers": [
      {
        "name": "app",
        "requests": {
          "cpu": "100m",
          "memory": "128Mi"
        },
        "limits": {
          "cpu": "200m",
          "memory": "256Mi"
        }
      }
    ],
    "total": {
      "cpuRequest": "100m",
      "cpuLimit": "200m",
      "memoryRequest": "128Mi",
      "memoryLimit": "256Mi"
    }
  },
  "metadata": {
    "requestId": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2023-06-21T10:30:00Z"
  }
}
```

#### Get Pod Failure Events
```http
GET /api/v1/pods/{namespace}/{podName}/failure-events
```

Returns analyzed failure events for a pod, categorizing issues and providing actionable insights.

**Example:**
```bash
curl http://k8s-cluster-agent.k8s-cluster-agent.svc.cluster.local/api/v1/pods/default/my-pod/failure-events
```

**Response:**
```json
{
  "data": {
    "podName": "my-pod",
    "namespace": "default",
    "totalEvents": 15,
    "failureEvents": [
      {
        "type": "Warning",
        "reason": "CrashLoopBackOff",
        "message": "Back-off restarting failed container",
        "firstTimestamp": "2023-06-21T09:00:00Z",
        "lastTimestamp": "2023-06-21T10:25:00Z",
        "count": 10,
        "source": "kubelet/node-1",
        "category": "ContainerCrash",
        "severity": "critical",
        "isRecurring": true,
        "recurrenceRate": "6.0 times per hour",
        "timeSinceFirst": "1h25m",
        "possibleCauses": [
          "Application crash",
          "Missing dependencies",
          "Configuration error",
          "Container app exited with code 1",
          "Container app has restarted 10 times"
        ],
        "suggestedAction": "Examine container logs and fix application startup issues"
      },
      {
        "type": "Warning",
        "reason": "FailedScheduling",
        "message": "0/4 nodes are available: 3 insufficient memory, 1 node(s) had volume affinity conflict",
        "firstTimestamp": "2023-06-21T08:00:00Z",
        "lastTimestamp": "2023-06-21T08:30:00Z",
        "count": 5,
        "source": "default-scheduler",
        "category": "Scheduling",
        "severity": "critical",
        "isRecurring": true,
        "recurrenceRate": "10.0 times per hour",
        "timeSinceFirst": "2h25m",
        "possibleCauses": [
          "Insufficient resources",
          "Node selector mismatch",
          "Affinity rules",
          "Taints not tolerated"
        ],
        "suggestedAction": "Check node resources and scheduling constraints"
      }
    ],
    "eventCategories": {
      "ContainerCrash": 1,
      "Scheduling": 1
    },
    "criticalEvents": 2,
    "warningEvents": 0,
    "mostRecentIssue": {
      "reason": "CrashLoopBackOff",
      "message": "Back-off restarting failed container",
      "lastTimestamp": "2023-06-21T10:25:00Z"
    },
    "ongoingIssues": [
      "CrashLoopBackOff: Back-off restarting failed container"
    ],
    "podPhase": "Running",
    "podStatus": "CrashLoopBackOff"
  },
  "metadata": {
    "requestId": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2023-06-21T10:30:00Z"
  }
}
```

**Event Categories:**
- `Scheduling`: Pod scheduling failures
- `ImagePull`: Image pull errors (ImagePullBackOff, ErrImagePull)
- `ContainerCrash`: Container crashes and restarts (CrashLoopBackOff, BackOff)
- `Volume`: Volume attachment and mounting issues
- `Resource`: Resource limits exceeded (OOMKilled, Evicted)
- `Probe`: Liveness/readiness probe failures
- `Network`: Network connectivity issues
- `Other`: Uncategorized failures

**Features:**
- **Intelligent Filtering**: Identifies problematic events (errors, warnings, high-frequency events)
- **Severity Classification**: Categorizes events as critical, warning, or info
- **Recurrence Analysis**: Tracks event frequency and calculates recurrence rates
- **Actionable Insights**: Provides possible causes and suggested actions for each failure type
- **Ongoing Issues**: Highlights problems that occurred in the last 5 minutes

#### Get Namespace Error Analysis
```http
GET /api/v1/namespace/{namespace}/error
```

Analyzes all pods in a namespace for common issues, filtering to only Deployments and StatefulSets (excludes Jobs).

**Example:**
```bash
curl http://k8s-cluster-agent.k8s-cluster-agent.svc.cluster.local/api/v1/namespace/default/error
```

**Response:**
```json
{
  "data": {
    "namespace": "default",
    "analysisTime": "2023-06-21T10:30:00Z",
    "totalPodsAnalyzed": 10,
    "problematicPodsCount": 3,
    "healthyPodsCount": 7,
    "restartThresholdUsed": 5,
    "summary": [
      {
        "issueType": "CrashLoopBackOff",
        "count": 2,
        "description": "Pods in crash loop backoff",
        "affectedPods": ["app-1-xyz", "app-2-abc"]
      },
      {
        "issueType": "HighRestarts",
        "count": 1,
        "description": "Pods with excessive restart counts",
        "affectedPods": ["worker-3-def"]
      }
    ],
    "problematicPods": [
      {
        "name": "app-1-xyz",
        "namespace": "default",
        "ownerKind": "Deployment",
        "ownerName": "app-1",
        "phase": "Running",
        "restartCount": 15,
        "age": "2h30m",
        "issues": [
          {
            "type": "HighRestarts",
            "description": "Pod has restarted 15 times (threshold: 5)",
            "severity": "critical",
            "details": "main: 15 restarts"
          },
          {
            "type": "CrashLoopBackOff",
            "description": "Container main is in CrashLoopBackOff state",
            "severity": "critical",
            "details": "Back-off restarting failed container"
          }
        ],
        "recentEvents": [
          {
            "type": "Warning",
            "reason": "BackOff",
            "message": "Back-off restarting failed container",
            "lastTimestamp": "2023-06-21T10:25:00Z",
            "count": 50
          }
        ]
      }
    ],
    "criticalIssuesCount": 5,
    "warningIssuesCount": 1
  },
  "metadata": {
    "requestId": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2023-06-21T10:30:00Z"
  }
}
```

**Issue Types Detected:**
- `HighRestarts`: Pods with restart count above threshold (configurable via POD_RESTART_THRESHOLD)
- `Pending`: Pods stuck in pending state for more than 5 minutes
- `Failed`: Pods in failed state
- `CrashLoopBackOff`: Containers repeatedly crashing
- `ImagePullError`: Image pull failures (ImagePullBackOff, ErrImagePull)
- `ResourceConstraints`: Insufficient resources for scheduling
- `Unschedulable`: Pods that cannot be scheduled

**Features:**
- **Configurable Thresholds**: Restart threshold configurable via environment variable
- **Owner Filtering**: Only analyzes pods owned by Deployments and StatefulSets
- **Issue Aggregation**: Groups issues by type with affected pod lists
- **Recent Events**: Includes recent warning events for problematic pods
- **Actionable Insights**: Provides specific details about each issue

#### Get Node Utilization
```http
GET /api/v1/nodes/{nodeName}/utilization
```

Returns current resource utilization for a node (requires metrics server).

**Example:**
```bash
curl http://k8s-cluster-agent.k8s-cluster-agent.svc.cluster.local/api/v1/nodes/node-1/utilization
```

**Response:**
```json
{
  "data": {
    "nodeName": "node-1",
    "cpuUsage": "500m",
    "cpuCapacity": "4000m",
    "cpuPercentage": 12.5,
    "memoryUsage": "2Gi",
    "memoryCapacity": "8Gi",
    "memoryPercentage": 25.0,
    "timestamp": "2023-06-21T10:30:00Z"
  },
  "metadata": {
    "requestId": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2023-06-21T10:30:00Z"
  }
}
```

### Error Responses

All errors follow a consistent format:

```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Pod not found",
    "details": "Pod 'my-pod' not found in namespace 'default'"
  },
  "metadata": {
    "requestId": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2023-06-21T10:30:00Z"
  }
}
```

Error codes:
- `BAD_REQUEST`: Invalid request parameters
- `RESOURCE_NOT_FOUND`: Pod or node not found
- `SERVICE_UNAVAILABLE`: Metrics server not available
- `REQUEST_TIMEOUT`: Request exceeded 500ms timeout
- `INTERNAL_ERROR`: Unexpected server error

## Development

### Prerequisites

- Go 1.21+
- Docker
- Make
- Access to a Kubernetes cluster

### Setup

1. Clone the repository:
```bash
git clone https://github.com/sumandas0/k8s-cluster-agent.git
cd k8s-cluster-agent
```

2. Install dependencies:
```bash
make deps
go mod download
```

3. Run tests:
```bash
make test
```

4. Run all checks:
```bash
make check  # Runs tidy, lint, and tests
```

### Common Commands

```bash
# Building
make build              # Build local binary
make docker-build       # Build Docker image
make run               # Run locally (requires kubeconfig)

# Testing & Quality
make test              # Run unit tests with coverage
make test-integration  # Run integration tests
make lint             # Run golangci-lint
make coverage         # View HTML coverage report
make check            # Run all checks (tidy, lint, test)

# Deployment
make deploy-dev        # Deploy to development
make deploy-prod       # Deploy to production

# Development
make generate-mocks    # Generate mocks from interfaces
make tidy             # Clean up go.mod dependencies
```

### Project Structure

```
k8s-cluster-agent/
├── cmd/agent/             # Application entry point
├── internal/              # Private application code
│   ├── core/             # Business logic
│   │   ├── domain/       # Domain models and interfaces
│   │   ├── ports/        # Service interfaces
│   │   └── services/     # Service implementations
│   ├── transport/        # HTTP handlers and middleware
│   │   └── http/         # REST API implementation
│   └── kubernetes/       # Kubernetes client
├── deployments/          # Kubernetes manifests
│   ├── base/            # Base manifests
│   └── overlays/        # Environment-specific overlays
│       ├── development/  # Dev configuration
│       └── production/   # Prod configuration
├── build/               # Docker and build files
├── docs/                # Documentation
└── CLAUDE.md            # AI assistant instructions
```

## Configuration

The agent is configured through environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `LOG_FORMAT` | `json` | Log format (json, text) |
| `K8S_TIMEOUT` | `30s` | Kubernetes API timeout |
| `READ_TIMEOUT` | `10s` | HTTP server read timeout |
| `WRITE_TIMEOUT` | `10s` | HTTP server write timeout |
| `POD_RESTART_THRESHOLD` | `5` | Restart count threshold for namespace error analysis |

## Security

### RBAC Permissions

The agent requires minimal permissions:
- `get`, `list` on `pods` (all namespaces)
- `get`, `list` on `events` (all namespaces)
- `get` on `nodes`
- `get` on `nodes/metrics`
- `get`, `list` on `deployments`, `statefulsets` (apps API group)

### Container Security

- Runs as non-root user (UID 65534)
- Read-only root filesystem
- No privilege escalation
- All capabilities dropped

## Monitoring

### Health Checks

- `/healthz` - Liveness probe
- `/readyz` - Readiness probe

### Logging

The agent uses structured logging with slog. All requests are logged with:
- Request ID for tracing
- Response time
- Status code
- Error details (if any)

### Metrics

In production, logs can be parsed and aggregated for monitoring request latency, error rates, and resource usage patterns.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details. 