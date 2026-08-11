# 3D API

## Purpose

This repository contains the HTTP API for the 3D/STL quality-check service.
It currently creates and retrieves in-memory jobs. It will eventually
orchestrate uploads and asynchronous processing performed by the separate
.NET processor.

Do not duplicate STL parsing or geometry validation in this service.

## Technology

- Go
- Standard library `net/http`
- Standard library `testing` and `httptest`
- No HTTP framework

## Architecture

Executable entry points live under `cmd/`.

The OpenAPI contract lives in `api/openapi.yaml` and is embedded into the
binary. Update it whenever an HTTP endpoint or DTO changes.

Application code lives under `internal/`:

- `internal/http` contains routing, handlers, and HTTP DTOs.
- `internal/jobs` contains the job domain, service, and repository.
- `internal/response` contains shared JSON response helpers.

Keep HTTP-specific concerns out of the jobs package. Handlers should call the
service rather than accessing repository storage directly.

Use interfaces only where they provide a useful boundary, such as the job
repository. Avoid unnecessary layers and dependency-injection frameworks.

## Coding guidelines

- Prefer simple, idiomatic Go over clever abstractions.
- Use the standard library where practical and keep dependencies minimal.
- Keep functions and packages focused.
- Use explicit request and response DTOs for HTTP JSON.
- Return stable client-facing errors without internal details.
- Protect shared mutable state used by concurrent HTTP requests.
- Run `gofmt` on changed Go files.
- Do not add AWS, databases, authentication, queues, or external services
  unless explicitly requested.

## Testing

Every new behavior should have automated tests where practical.

Run before considering a change complete:

    go test ./...
    go vet ./...

Use the race detector when changing concurrent code:

    go test -race ./...

Tests must be deterministic and must not require network access.

## Changes

Do not make unrelated refactors while implementing a requested feature.
Keep changes small and easy to review.
