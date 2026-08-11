# 3d-api

Initial HTTP API for the 3D/STL quality-check service. Jobs are currently kept in memory and are lost when the process stops.

## Run

Go 1.26.5 or later is required. From the repository root, start the API with:

```bash
go run ./cmd/api
```

The API listens on port `8080`. Set `PORT` to override it:

```bash
PORT=9000 go run ./cmd/api
```

Stop the server with `Ctrl+C`. Jobs use in-memory storage and are lost when
the server stops.

## Endpoints

Check service health:

```bash
curl http://localhost:8080/health
```

Create a job:

```bash
curl -i -X POST http://localhost:8080/jobs \
  -H 'Content-Type: application/json' \
  -d '{"fileName":"rotor.stl","fileSizeBytes":117884}'
```

Retrieve a job using the `id` returned by the create request:

```bash
curl http://localhost:8080/jobs/JOB_ID
```

## Run tests

Run the complete test suite from the repository root:

```bash
go test ./...
```

Run static analysis:

```bash
go vet ./...
```

When changing concurrent repository code, also run the tests with Go's race
detector:

```bash
go test -race ./...
```

## Pre-commit hook

This repository includes a hook at `.githooks/pre-commit` that runs
`go test ./...` before every commit. Enable repository-managed hooks once per
clone with:

```bash
git config core.hooksPath .githooks
```

A commit is stopped if the tests fail.
