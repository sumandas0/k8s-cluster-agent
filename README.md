# K8s Cluster Agent

A lightweight in-cluster Kubernetes agent that provides read-only access to pod and node information through a RESTful API.

## Features

- **Pod Information**: Get detailed pod information including scheduling details and resource requirements
- **Node Utilization**: Retrieve real-time node resource utilization metrics
- **Secure by Default**: Minimal RBAC permissions, read-only access
- **High Performance**: 500ms request timeout, efficient resource usage
- **Production Ready**: Health checks, structured logging, graceful shutdown

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

3. Deploy to Kubernetes (development):
```bash
make deploy-dev
```

For production deployment:
```bash
make deploy-prod
```

### Manual Installation

Apply the Kubernetes manifests directly:
```bash
kubectl apply -k deployments/overlays/development
```

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

Returns scheduling-specific information for a pod.

**Example:**
```bash
curl http://k8s-cluster-agent.k8s-cluster-agent.svc.cluster.local/api/v1/pods/default/my-pod/scheduling
```

**Response:**
```json
{
  "data": {
    "nodeName": "node-1",
    "schedulerName": "default-scheduler",
    "nodeSelector": {
      "kubernetes.io/os": "linux"
    },
    "tolerations": [],
    "affinity": null,
    "priority": 0,
    "priorityClassName": ""
  },
  "metadata": {
    "requestId": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2023-06-21T10:30:00Z"
  }
}
```

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

4. Run linter:
```bash
make lint
```

### Building

Build the binary:
```bash
make build
```

Build Docker image:
```bash
make docker-build
```

### Testing

Run unit tests:
```bash
make test
```

Run integration tests:
```bash
make test-integration
```

View coverage report:
```bash
make coverage
```

### Project Structure

```
k8s-cluster-agent/
├── cmd/agent/          # Application entry point
├── internal/           # Private application code
│   ├── core/          # Business logic
│   ├── transport/     # HTTP handlers and middleware
│   └── kubernetes/    # Kubernetes client
├── deployments/       # Kubernetes manifests
├── build/            # Docker and build files
└── docs/             # Documentation
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

## Security

### RBAC Permissions

The agent requires minimal permissions:
- `get`, `list` on `pods` (all namespaces)
- `get` on `nodes`
- `get` on `nodes/metrics`

### Container Security

- Runs as non-root user (UID 65534)
- Read-only root filesystem
- No privilege escalation
- All capabilities dropped

## Monitoring

### Health Checks

- `/healthz` - Liveness probe
- `/readyz` - Readiness probe

### Metrics

The agent logs all requests with duration and status code. In production, these can be scraped for monitoring.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details. 