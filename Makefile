.PHONY: all doc-start lint test dummy-build install-tools

OS := $(if $(GOOS),$(GOOS),$(shell go env GOOS))
ARCH := $(if $(GOARCH),$(GOARCH),$(shell go env GOARCH))
BUILD_STRING := $(OS)-$(ARCH)

all: lint test build
	@echo "Finished successfully"

doc-serve:
	pkgsite -open

install-tools:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.62.2
	go install golang.org/x/pkgsite/cmd/pkgsite@latest
	go install github.com/miniscruff/changie@latest
	go install github.com/goreleaser/goreleaser@latest

lint:
	golangci-lint run

test:
	go test -v -cover ./internal/... ./pkg/...

build:
	go build -o bin/$(BUILD_STRING)/go-homebank-csv cmd/go-homebank-csv/main.go

clean:
	rm -rf bin
	rm -rf dist
