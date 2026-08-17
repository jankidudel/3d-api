# 3d-api

HTTP API for the 3D/STL quality-check service. Production job metadata and processor reports are stored in S3 under `jobs/{jobId}/`.

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
roles. The API process requires `s3:GetObject` and `s3:PutObject` access to the
configured bucket's `jobs/*` objects (`HeadObject` is authorized by
`s3:GetObject`).

Stop the server with `Ctrl+C`. Job metadata remains in S3 when the local server stops.

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

The response includes a presigned S3 POST form valid for 15 minutes. Submit every
returned field followed by the file as multipart form data. For example, if the
response is saved as `job.json`:

```bash
curl $(jq -r '.upload.fields | to_entries[] | "-F \(.key)=\(.value|@sh)"' job.json) \
  -F 'file=@rotor.stl' "$(jq -r .upload.url job.json)"
```

The uploaded object key is `jobs/{job-id}/input.stl`. For browser uploads, the
S3 bucket must have a CORS rule that allows `POST` from the frontend's origin.
Do not make the bucket public; the presigned URL grants temporary permission
for this specific upload.

Uploads are limited to 25 MiB (26,214,400 bytes). Larger create-job requests
return HTTP 413 before an upload form is issued. The signed S3 POST policy also
contains a `content-length-range` condition, so S3 rejects a form body larger
than the size declared when the job was created. The processor independently
rejects actual objects larger than 25 MiB before downloading them.

Retrieve a job using the `id` returned by the create request:

```bash
curl http://localhost:8080/api/v1/jobs/JOB_ID
```

The API reads `jobs/{job-id}/metadata.json`, reports `processing` once the input
object exists, and returns `completed` with the JSON report once
`jobs/{job-id}/report.json` exists.

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

Tests cover the jobs service validation and retrieval behavior, the S3-backed
production repository, the thread-safe in-memory test repository, concurrent
repository access, HTTP routing, JSON validation, health checks, error responses,
and the Swagger/OpenAPI endpoints.

## Processing flow

The frontend calls `GET /api/v1/jobs/{id}` to refresh a job. The API returns
`processing` while the input exists without a report:

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
  "report": {}
}
```

Statuses are `created`, `processing`, `completed`, and `failed`. On the final SQS
attempt, the processor stores `jobs/{jobId}/failure.json`; the API returns its
safe structured error when no completed report exists.

## Package for Lambda

Build the custom-runtime `provided.al2023` artifact with:

```bash
make package
```

This runs all checks, cross-compiles `cmd/lambda` for Linux x86_64, and creates
the ignored `artifacts/3d-api.zip` containing the required `bootstrap` executable.

## Pre-commit hook

This repository includes a hook at `.githooks/pre-commit` that runs
`make check` before every commit. Enable repository-managed hooks once per
clone with:

```bash
git config core.hooksPath .githooks
```

A commit is stopped if Go files need formatting, vet or linting fails, or
tests fail.
