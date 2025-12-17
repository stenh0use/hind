# Versioning for Go CLI
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)
LDFLAGS := -X github.com/stenh0use/hind/pkg/cmd/hind/version.Version=$(VERSION) \
		   -X github.com/stenh0use/hind/pkg/cmd/hind/version.Commit=$(COMMIT)

# Go CLI build
.PHONY: hind-cli
hind-cli:
	go build -ldflags "$(LDFLAGS)" -o bin/hind

.PHONY: build
build: hind-cli

.PHONY: test
test:
	@ go fmt ./...
	@ go vet ./...
	@ go test ./...
