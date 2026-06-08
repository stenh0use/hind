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

.PHONY: test-e2e-clean
test-e2e-clean: hind-cli
	@ HIND_E2E_BIN=$(PWD)/bin/hind go test -tags e2e ./test/e2e/cli -run "TestE2E_PreflightCleanup_RemovesRunningCluster|TestE2E_PreflightCleanup_HandlesMissingCluster|TestE2E_PreflightCleanup_IsIdempotent" -v -count=1

.PHONY: test-e2e
test-e2e: test-e2e-clean
	@ HIND_E2E_BIN=$(PWD)/bin/hind go test -tags e2e ./test/e2e/cli -v -count=1

.PHONY: test-e2e-one
test-e2e-one: hind-cli
	@if [ -z "$(TEST)" ]; then echo "TEST is required (example: make test-e2e-one TEST=TestE2E_Lifecycle_StartStopStartRmStop)"; exit 1; fi
	@ HIND_E2E_BIN=$(PWD)/bin/hind go test -tags e2e ./test/e2e/cli -run "$(TEST)" -v -count=1
