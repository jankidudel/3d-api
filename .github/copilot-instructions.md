# GitHub Copilot Instructions

Follow the repository architecture and development guidelines documented in
`AGENTS.md`.

When generating Go:

- Prefer simple, idiomatic, maintainable implementations.
- Use the standard library and avoid adding dependencies unless necessary.
- Keep HTTP concerns in `internal/http` and domain behavior in `internal/jobs`.
- Use explicit JSON DTOs rather than exposing domain models from handlers.
- Keep `api/openapi.yaml` synchronized with HTTP endpoints and DTOs.
- Keep the in-memory repository safe for concurrent access.
- Do not implement STL parsing or geometry validation in Go.
- Add or update tests when behavior changes.
- Do not modify unrelated code.
- Run `gofmt` on changed Go files.

Before completing implementation work, ensure:

    go test ./...
    go vet ./...

When concurrent code changes, also run:

    go test -race ./...
