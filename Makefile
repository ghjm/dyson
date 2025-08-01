PROGRAM := dyson
PLATFORMS := linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64
PROGRAM_DEPS := Makefile data.yml go.mod go.sum main.go $(call go_deps,main.go)
SHELL := /bin/bash

# Get commit hash from git
VERSION_FLAGS := $(shell git rev-parse HEAD)
ifneq ($(VERSION_FLAGS),)
	VERSION_FLAGS := -ldflags="-X 'main.gitCommit=$(VERSION_FLAGS)'"
endif

# go_deps calculates the Go dependencies required to compile a given source file
define go_deps
$(shell find $(shell go list -f '{{.Dir}}' -deps $(1) | grep "^$$PWD") -name '*.go' | grep -v '_test.go$$' | grep -v '_gen.go$$')
endef

# binary_path calculates the bin path for a given platform
define binary_path
$(if $(findstring windows,$(1)),bin/$(PROGRAM)-$(subst /,-,$(1)).exe,bin/$(PROGRAM)-$(subst /,-,$(1)))
endef

BINARIES := $(foreach plat,$(PLATFORMS),$(call binary_path,$(plat)))

$(PROGRAM): $(PROGRAM_DEPS)
	go build -o $(PROGRAM) $(VERSION_FLAGS) main.go

.PHONY: bin
bin: $(BINARIES)

bin/$(PROGRAM)-%: $(PROGRAM_DEPS)
	@echo "Building bin/$(PROGRAM)-$*"
	@base=$* ; \
	[[ "$$base" == *.exe ]] && base=$${base%.exe} ; \
	GOOS=$$(echo $$base | cut -d- -f1) ; \
	GOARCH=$$(echo $$base | cut -d- -f2) ; \
	GOOS=$$GOOS GOARCH=$$GOARCH \
	go build -o $@ $(VERSION_FLAGS) main.go

.PHONY: all
all: $(PROGRAM) bin

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
	rm -f $(PROGRAM)
	rm -rf bin

