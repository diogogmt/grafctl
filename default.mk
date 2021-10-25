BUILD_TAGS ?=
BIN_NAME ?= grafctl
GO_LDFLAGS ?=
GO_GCFLAGS ?= -e
BIN_DIR ?= ./
define build
	go build \
		-tags='$(BUILD_TAGS)' \
		-gcflags='$(GO_GCFLAGS)' \
		-ldflags='$(GO_LDFLAGS)' \
		-o $(BIN_DIR)/$(BIN_NAME) ./cmd/grafctl
endef

define install
	go install \
		-tags='$(BUILD_TAGS)' \
		-gcflags='$(GO_GCFLAGS)' \
		-ldflags='$(GO_LDFLAGS)' \
		./cmd/grafctl
endef

.PHONY: build
## build: builds grafctl
build:
	$(call build)

.PHONY: install
## install: installs grafctl
install:
	$(call install)

## imports: runs goimports
imports:
	goimports -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")

## lint: runs golint
lint:
	golint ./...

## test: runs go test
test:
	go test ./...

## vet: runs go vet
vet:
	go vet ./...

## staticcheck: runs staticcheck
staticcheck:
	staticcheck $(shell go list ./...)


## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'