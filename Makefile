DIST_DIR ?= ./dist
EXE = $(DIST_DIR)/toe
GO ?= go

.PHONY: all install build format check test pre-commit coverage clean

all: build

install: test
	$(GO) install github.com/kode4food/toe/cmd/toe

build: test
	@mkdir -p $(DIST_DIR)
	@rm -f $(EXE)
	$(GO) build -o $(EXE) ./cmd/toe

format:
	$(GO) run golang.org/x/tools/cmd/goimports@latest -w .
	$(GO) fix ./...

check:
	$(GO) vet ./...
	$(GO) run honnef.co/go/tools/cmd/staticcheck@latest ./...

test: check
	$(GO) test ./...

pre-commit: format test

coverage:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -func=coverage.out

clean:
	@rm -f $(EXE)
	@rmdir $(DIST_DIR)
