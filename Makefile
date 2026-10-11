DIST_DIR ?= ./dist
EXE = $(DIST_DIR)/toe
GO ?= go
PYTHON ?= python3

.PHONY: all install build format check test pre-commit coverage cheatsheet clean

all: build

install: test
	$(GO) install github.com/kode4food/toe/cmd/toe

build: test
	@mkdir -p $(DIST_DIR)
	@rm -f $(EXE)
	$(GO) build -o $(EXE) ./cmd/toe

format:
	$(GO) tool goimports -w .
	$(GO) fix ./...

check:
	$(GO) vet ./...
	$(GO) tool staticcheck ./...

test: check
	$(GO) test ./...

pre-commit: format test cheatsheet

coverage:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -func=coverage.out

cheatsheet:
	@venv=$$(mktemp -d) && \
		trap 'rm -rf "$$venv"' EXIT && \
		$(PYTHON) -m venv "$$venv" && \
		"$$venv"/bin/pip install -q reportlab && \
		"$$venv"/bin/python docs/cheatsheet.py

clean:
	@rm -f $(EXE)
	@rmdir $(DIST_DIR)
