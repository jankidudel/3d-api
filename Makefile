.PHONY: fmt lint test test-race vet check

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
