VERSION := $(shell cat VERSION)
CLI_LDFLAGS := -ldflags "-X main.version=$(VERSION)"
BACKEND_LDFLAGS := -ldflags "-X main.Version=$(VERSION)"

.PHONY: build build-cli build-backend test test-cli test-backend vet lint frontend clean

## Build everything
build: build-cli build-backend

build-cli:
	go build $(CLI_LDFLAGS) -o bin/wizards-qa ./cmd

build-backend:
	cd web/backend && go build $(BACKEND_LDFLAGS) -o ../../bin/wizards-qa-server .

## Run all tests
test: test-cli test-backend

test-cli:
	go test ./...

test-backend:
	cd web/backend && go test ./...

## Vet
vet:
	go vet ./...
	cd web/backend && go vet ./...

## Frontend
frontend:
	cd web/frontend && npm run build

## Full validation (matches CI)
validate: vet test frontend

## Clean build artifacts
clean:
	rm -rf bin/ web/frontend/dist/
