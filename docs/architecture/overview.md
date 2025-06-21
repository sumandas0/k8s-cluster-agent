# K8s Cluster Agent Architecture Overview

## Overview

The K8s Cluster Agent is a lightweight, read-only service that runs inside a Kubernetes cluster and provides a RESTful API for querying pod and node information. It follows clean architecture principles with clear separation between business logic and transport layers.

## Architecture Principles

1. **Clean Architecture**: Core business logic is independent of transport mechanisms
2. **Dependency Injection**: Using interfaces for testability and flexibility
3. **Security First**: Minimal RBAC permissions, read-only access
4. **Performance**: 500ms request timeout, efficient resource usage
5. **Observability**: Structured logging, health checks, metrics-ready

## Component Structure

```
┌─────────────────────────────────────────────────────────────┐
│                        HTTP Client                           │
└────────────────────┬────────────────────────────────────────┘
                     │ REST API
┌────────────────────▼────────────────────────────────────────┐
│                   Transport Layer (HTTP)                     │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐    │
│  │   Handlers   │ │  Middleware  │ │    Router        │    │
│  └──────────────┘ └──────────────┘ └──────────────────┘    │
└────────────────────┬────────────────────────────────────────┘
                     │ Interfaces
┌────────────────────▼────────────────────────────────────────┐
│                      Core Layer                              │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐    │
│  │  PodService  │ │ NodeService  │ │     Models       │    │
│  └──────────────┘ └──────────────┘ └──────────────────┘    │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                 Kubernetes Client Layer                      │
│  ┌──────────────┐ ┌──────────────────────────────────┐     │
│  │  K8s Client  │ │    Metrics Client                │     │
│  └──────────────┘ └──────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

## Key Components

### Transport Layer
- **HTTP Server**: Chi router-based HTTP server
- **Handlers**: Convert HTTP requests to service calls
- **Middleware**: Logging, recovery, timeout, request ID
- **Response Helpers**: Consistent error and success responses

### Core Layer
- **Services**: Business logic for pod and node operations
- **Models**: Data structures for API responses
- **Interfaces**: Service contracts for dependency injection
- **Errors**: Domain-specific error definitions

### Kubernetes Layer
- **Client Initialization**: In-cluster configuration
- **Clientsets**: Standard and metrics clients
- **Error Handling**: Graceful degradation when metrics unavailable

## Data Flow

1. **Request Reception**: HTTP request arrives at the router
2. **Middleware Processing**: Request ID, logging, timeout applied
3. **Handler Invocation**: URL parameters extracted, validation performed
4. **Service Call**: Handler calls appropriate service method
5. **Kubernetes API**: Service queries Kubernetes API/metrics
6. **Response Formation**: Data transformed to response models
7. **Response Delivery**: JSON response with metadata sent to client

## Security Model

### RBAC Permissions
- `get`, `list` on `pods` (cluster-wide)
- `get` on `nodes`
- `get` on `nodes/metrics`

### Container Security
- Non-root user (UID 65534)
- Read-only root filesystem
- No privilege escalation
- All capabilities dropped

### Network Security
- No external dependencies
- Service-to-service communication only
- Optional NetworkPolicy for egress control

## Performance Characteristics

- **Request Timeout**: 500ms hard limit
- **Resource Usage**: 50m CPU, 64Mi memory (typical)
- **Startup Time**: < 5 seconds
- **Concurrent Requests**: Limited by CPU/memory
- **Kubernetes API Timeout**: 30 seconds

## Deployment Model

- **Namespace**: Dedicated namespace for isolation
- **Replicas**: 2 (development: 1, production: 2-10 with HPA)
- **Service**: ClusterIP for internal access
- **Health Checks**: Liveness and readiness probes
- **Autoscaling**: HPA based on CPU/memory (production)

## Observability

### Logging
- Structured JSON logging (configurable to text)
- Request/response logging with duration
- Error logging with stack traces
- Configurable log levels

### Metrics (Future)
- HTTP request duration histogram
- HTTP request counter by endpoint/status
- Kubernetes API call metrics
- Go runtime metrics

### Health Checks
- `/healthz`: Basic liveness check
- `/readyz`: Readiness including dependency checks

## Extension Points

1. **New Resource Types**: Add new services in core layer
2. **Transport Protocols**: Add gRPC alongside HTTP
3. **Caching**: Add caching layer for frequently accessed data
4. **Webhooks**: Add webhook support for real-time updates
5. **Multi-cluster**: Extend to support multiple clusters

## Technology Stack

- **Language**: Go 1.21+
- **HTTP Router**: Chi v5
- **Kubernetes Client**: client-go v0.28.4
- **Logging**: slog (structured logging)
- **Testing**: Standard library + fake clients
- **Container**: Alpine-based, minimal image

## Best Practices

1. **Error Handling**: Always wrap errors with context
2. **Context Propagation**: Pass context through all layers
3. **Resource Cleanup**: Use defer for cleanup operations
4. **Testing**: Unit tests with fake clients, integration tests
5. **Documentation**: Code comments, API documentation
6. **Security**: Principle of least privilege, defense in depth 