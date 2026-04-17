# Calebsons Go Cloud-Native — Distributed Task Queue

## Overview
A distributed task queue system with worker pools, gRPC communication, and Kubernetes-ready deployments.

## Tech Stack
- Go
- gRPC
- Redis / NATS
- Kubernetes

## Features
- Distributed workers
- Task scheduling
- Retry logic
- gRPC API
- Horizontal scaling

## Architecture
```mermaid
flowchart TD
    PRODUCER[Producers / Services] --> QUEUE[Redis / NATS Broker]
    QUEUE --> WORKER1[Go Worker 1]
    QUEUE --> WORKER2[Go Worker 2]
    WORKER1 --> STATE[Task State Store]
    WORKER2 --> STATE
    INFRA[Kubernetes Deployments] -.-> WORKER1
    INFRA -.-> WORKER2
```

## Setup
    go mod tidy
    go run cmd/server/main.go

## Deployment
- Kubernetes
- Helm (optional)

## Roadmap
- Add dashboard UI
- Add delayed jobs
