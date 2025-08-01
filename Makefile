SHELL := /bin/bash

VERSION_FLAGS := $(shell git rev-parse HEAD)
ifneq ($(VERSION_FLAGS),)
	VERSION_FLAGS := -ldflags="-X 'main.gitCommit=$(VERSION_FLAGS)'"
endif

PLATFORMS := linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64

dyson: main.go $(shell find pkg/dyson -type f -name '*.go')
	go build -o dyson $$VERSION_FLAGS main.go

.PHONY: bin
.ONESHELL:
bin:
	@for PLAT in $(PLATFORMS); do
	    echo Building $$PLAT...
	    IFS='/' read -ra PATH_PARTS <<< $$PLAT
	    OS=$${PATH_PARTS[0]}
	    ARCH=$${PATH_PARTS[1]}
	    if [ "$$OS" == "windows" ]; then
	        SUFFIX=".exe"
	    else
	        SUFFIX=""
	    fi
	    GOOS=$$OS GOARCH=$$ARCH go build -o bin/dyson-$$OS-$$ARCH$$SUFFIX $(VERSION_FLAGS) main.go
	done

.PHONY: lint
lint:
	@golangci-lint run

.PHONY: test
test:
	@go test ./... -count=1

.PHONY: fmt
fmt:
	@go fmt ./...

.PHONY: check-fmt
check-fmt:
	@if [[ $$(gofmt -l .) ]]; then echo Code needs to be formatted; exit 1; fi

.PHONY: clean
clean:
	rm -f dyson
	rm -rf bin

