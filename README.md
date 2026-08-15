# 3d-api

Initial HTTP API for the 3D/STL quality-check service. Jobs are currently kept in memory and are lost when the process stops.

## Run

Go 1.26.5 or later is required. Create your local configuration file once:

```bash
cp .env.example .env
```

Edit `.env` with your S3 bucket and AWS region. Set `AWS_PROFILE` only if that
named profile already exists in your local AWS configuration. Then start the
API from the repository root:

```bash
make run
```

The Makefile loads `.env` before running `go run ./cmd/api`. The `.env` file is
ignored by Git; `.env.example` documents the supported local settings without
containing credentials or environment-specific values.

This runs `go run ./cmd/api`.

The API listens on the `PORT` configured in `.env`, which defaults to `8080`
in the example file.

AWS credentials are loaded through the AWS SDK's standard credential chain,
including environment variables, shared AWS config/profile files, and IAM
roles. The API process requires permission to sign `s3:PutObject` requests for
the configured bucket.

Stop the server with `Ctrl+C`. Jobs use in-memory storage and are lost when
the server stops.

## Swagger UI

With the API running, open the interactive API documentation in a browser:

```text
http://localhost:8080/api/swagger/
```

Use **Try it out** in the Swagger UI to call the health and job endpoints.
The raw OpenAPI 3.1 document is available at:

```text
http://localhost:8080/api/v1/openapi.yaml
```

The UI loads its version-pinned Swagger UI assets from jsDelivr, so the
browser needs internet access to render the interactive page. The API and its
OpenAPI document do not require internet access.

## Format code

Go includes an official formatter, `gofmt`. Format every Go file with:

```bash
make fmt
```

Most Go IDE extensions can run `gofmt` automatically when a file is saved.
For VS Code, install the official Go extension and enable format on save.

## Endpoints

Check service health:

```bash
curl http://localhost:8080/api/v1/health
```

Create a job:

```bash
curl -i -X POST http://localhost:8080/api/v1/jobs \
  -H 'Content-Type: application/json' \
  -d '{"fileName":"rotor.stl","fileSizeBytes":117884}'
```

The response includes a presigned upload URL valid for 15 minutes. Upload the
file directly to S3 using the returned method and URL:

```bash
curl -X PUT --upload-file rotor.stl 'PRESIGNED_UPLOAD_URL'
```

The uploaded object key is `jobs/{job-id}/input.stl`. For browser uploads, the
S3 bucket must have a CORS rule that allows `PUT` from the frontend's origin.
Do not make the bucket public; the presigned URL grants temporary permission
for this specific upload.

Retrieve a job using the `id` returned by the create request:

```bash
curl http://localhost:8080/api/v1/jobs/JOB_ID
```

## Run tests

Run the complete test suite from the repository root:

```bash
make test
```

Run static analysis:

```bash
make vet
```

Run Staticcheck linting across all Go packages:

```bash
make lint
```

Staticcheck is pinned as a Go tool in `go.mod`, so contributors use the same
version without installing it globally. The first run may download the tool.

When changing concurrent repository code, also run the tests with Go's race
detector:

```bash
make test-race
```

Run formatting validation, vet, linting, and tests together with:

```bash
make check
```

Tests cover the jobs service validation and retrieval behavior, the
thread-safe in-memory repository, concurrent repository access, HTTP routing,
JSON validation, health checks, job creation/retrieval, error responses, and
the Swagger/OpenAPI endpoints.

## Future processing flow

Once asynchronous processing is implemented, the frontend
should poll `GET /api/v1/jobs/{id}` until the job reaches a terminal status. Use a
typed status rather than a separate `completed` boolean, for example:

```json
{
  "id": "...",
  "status": "processing"
}
```

followed by:

```json
{
  "id": "...",
  "status": "completed",
  "report": {},
  "downloadUrl": "https://..."
}
```

Expected statuses are `created`, `uploading`, `queued`, `processing`,
`completed`, and `failed`. The frontend should stop polling for `completed` or
`failed`, use a modest delay such as two seconds, and apply backoff if
processing takes longer. A presigned URL is temporary, so the API should
generate it when returning a completed result rather than persist it as job
state. WebSockets or server-sent events can be added later if polling becomes
too slow or expensive; polling is the simplest appropriate MVP design.

## Pre-commit hook

This repository includes a hook at `.githooks/pre-commit` that runs
`make check` before every commit. Enable repository-managed hooks once per
clone with:

```bash
git config core.hooksPath .githooks
```

A commit is stopped if Go files need formatting, vet or linting fails, or
tests fail.
