.PHONY: run fmt lint test test-race vet check package clean

ARTIFACT_DIR := $(CURDIR)/artifacts
LAMBDA_BINARY := $(ARTIFACT_DIR)/bootstrap
LAMBDA_PACKAGE := $(ARTIFACT_DIR)/3d-api.zip

run:
	@set -a; \
	if [ ! -f .env ]; then \
		echo ".env is missing. Run: cp .env.example .env"; \
		exit 1; \
	fi; \
	. ./.env; \
	set +a; \
	exec go run ./cmd/api

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

test:
	go test ./...

lint:
	go tool staticcheck ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

check:
	@unformatted="$$(find . -name '*.go' -not -path './vendor/*' -exec gofmt -l {} +)"; \
	if [ -n "$$unformatted" ]; then \
		echo "The following Go files need formatting:"; \
		echo "$$unformatted"; \
		echo "Run: make fmt"; \
		exit 1; \
	fi
	go vet ./...
	go tool staticcheck ./...
	go test ./...

package: check
	mkdir -p "$(ARTIFACT_DIR)"
	rm -f "$(LAMBDA_BINARY)" "$(LAMBDA_PACKAGE)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "$(LAMBDA_BINARY)" ./cmd/lambda
	cd "$(ARTIFACT_DIR)" && zip -q "$(LAMBDA_PACKAGE)" bootstrap
	rm -f "$(LAMBDA_BINARY)"

clean:
	rm -rf "$(ARTIFACT_DIR)"
