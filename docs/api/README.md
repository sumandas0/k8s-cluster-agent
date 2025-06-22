# K8s Cluster Agent API Documentation

This directory contains the auto-generated OpenAPI documentation for the K8s Cluster Agent API.

## Accessing the API Documentation

### Interactive Swagger UI
When the K8s Cluster Agent is running, you can access the interactive API documentation at:
```
http://localhost:8080/swagger/
```

### OpenAPI Specification Files
- **JSON Format**: [openapi.json](../swagger.json)
- **YAML Format**: [openapi.yaml](../swagger.yaml)

## Generating Documentation

The API documentation is generated from Go code annotations using [swaggo/swag](https://github.com/swaggo/swag).

To regenerate the documentation after making changes to API annotations:

```bash
make generate-openapi
```

Or manually:
```bash
swag init -g cmd/agent/main.go -o docs --parseInternal --parseDependency
```

## API Overview

The K8s Cluster Agent provides the following API endpoints:

### Pod Operations
- `GET /api/v1/pods/{namespace}/{podName}/describe` - Get comprehensive pod information
- `GET /api/v1/pods/{namespace}/{podName}/scheduling` - Get pod scheduling analysis
- `GET /api/v1/pods/{namespace}/{podName}/resources` - Get pod resource requirements
- `GET /api/v1/pods/{namespace}/{podName}/failure-events` - Get analyzed failure events
- `GET /api/v1/pods/{namespace}/{podName}/scheduling/explain` - Get detailed scheduling explanation
- `GET /api/v1/pods/{namespace}/{podName}/health-score` - Get pod health score

### Node Operations
- `GET /api/v1/nodes/{nodeName}/utilization` - Get node utilization metrics

### Namespace Operations
- `GET /api/v1/namespace/{namespace}/error` - Get namespace error analysis

### Cluster Operations
- `GET /api/v1/cluster/pod-issues` - Get cluster-wide pod issues dashboard

### Health Checks
- `GET /healthz` - Health check endpoint
- `GET /readyz` - Readiness check endpoint

## Response Format

All successful API responses follow this structure:
```json
{
  "data": {...},
  "metadata": {
    "requestId": "unique-request-id",
    "timestamp": "2023-01-01T00:00:00Z"
  }
}
```

Error responses follow this structure:
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message",
    "details": "Additional error details"
  },
  "metadata": {
    "requestId": "unique-request-id",
    "timestamp": "2023-01-01T00:00:00Z"
  }
}
```

## Authentication

Currently, the API does not require authentication. It relies on Kubernetes RBAC for access control.

## Rate Limiting

All API requests have a 500ms timeout to ensure high performance.