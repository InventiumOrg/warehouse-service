# Health Check Endpoints

## Overview

The warehouse service provides two health check endpoints for monitoring and orchestration:

## Endpoints

### `/healthz` - Liveness Probe

- **Purpose**: Basic liveness check
- **Method**: GET
- **Authentication**: None required
- **Response**: Always returns 200 OK if service is running

**Example Response:**

```json
{
  "status": "ok",
  "timestamp": "2024-01-15T10:30:00Z",
  "service": "warehouse-service"
}
```

### `/readyz` - Readiness Probe

- **Purpose**: Readiness check with dependency validation
- **Method**: GET
- **Authentication**: None required
- **Response**: 200 OK if ready, 503 if not ready

**Ready Response:**

```json
{
  "status": "ready",
  "timestamp": "2024-01-15T10:30:00Z",
  "service": "warehouse-service",
  "checks": {
    "database": "ok"
  }
}
```

**Not Ready Response:**

```json
{
  "status": "not ready",
  "timestamp": "2024-01-15T10:30:00Z",
  "service": "warehouse-service",
  "error": "database connection failed",
  "details": "connection refused"
}
```

## Usage in Kubernetes

### Probe Configuration Strategy

- **Liveness Probe**: Longer delay (30s) to avoid premature restarts during startup
- **Readiness Probe**: Shorter delay (5s) to serve traffic as soon as ready

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 7450
  initialDelaySeconds: 30 # Conservative: avoid killing during slow startup
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /readyz
    port: 7450
  initialDelaySeconds: 5 # Aggressive: start serving traffic quickly
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 2
```

### Why Different Delays?

| Probe Type    | Failure Action      | Delay Strategy     | Reasoning                               |
| ------------- | ------------------- | ------------------ | --------------------------------------- |
| **Liveness**  | Pod restart         | Conservative (30s) | Avoid expensive restarts during startup |
| **Readiness** | Remove from service | Aggressive (5s)    | Start serving traffic ASAP              |
