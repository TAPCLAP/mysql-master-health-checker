SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

BINARY ?= mysql-master-health-checker
MAIN ?= ./cmd/mysql-master-health-checker
LDFLAGS ?= -s -w
BUILD_FLAGS ?= -trimpath -buildvcs=false

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet.
	go vet ./...

.PHONY: test
test: fmt vet ## Run unit tests.
	go test ./... -coverprofile cover.out

.PHONY: test-integration
test-integration: fmt vet ## Run integration tests (requires Docker).
	go test -tags=integration ./... -count=1 -timeout=10m -coverprofile cover-integration.out

.PHONY: lint
lint: golangci-lint ## Run golangci-lint.
	"$(GOLANGCI_LINT)" run

.PHONY: lint-config
lint-config: golangci-lint ## Verify golangci-lint configuration.
	"$(GOLANGCI_LINT)" config verify

##@ Build

.PHONY: build
build: ## Build local binary.
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags="$(LDFLAGS)" -o bin/$(BINARY) $(MAIN)

.PHONY: build-linux-amd64
build-linux-amd64: ## Build statically linked linux/amd64 binary.
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -ldflags="$(LDFLAGS)" -o artifacts/$(BINARY)-linux-amd64 $(MAIN)

.PHONY: build-linux-arm64
build-linux-arm64: ## Build statically linked linux/arm64 binary.
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -ldflags="$(LDFLAGS)" -o artifacts/$(BINARY)-linux-arm64 $(MAIN)

.PHONY: build-release
build-release: ## Build release binaries for linux amd64 and arm64.
	mkdir -p artifacts
	$(MAKE) build-linux-amd64 build-linux-arm64

##@ Dependencies

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p "$(LOCALBIN)"

GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.11.4

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT)

$(GOLANGCI_LINT): $(LOCALBIN)
	@[ -f "$(GOLANGCI_LINT)-$(GOLANGCI_LINT_VERSION)" ] || { \
		GOBIN="$(LOCALBIN)" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
		mv "$(LOCALBIN)/golangci-lint" "$(GOLANGCI_LINT)-$(GOLANGCI_LINT_VERSION)"; \
	}
	@ln -sf "$$(realpath "$(GOLANGCI_LINT)-$(GOLANGCI_LINT_VERSION)")" "$(GOLANGCI_LINT)"
