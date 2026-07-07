STEAMPIPE_INSTALL_DIR ?= ~/.steampipe
BUILD_TAGS = netgo

.PHONY: install test fmt vet hooks

install:
	go build -o $(STEAMPIPE_INSTALL_DIR)/plugins/local/turbopuffer/turbopuffer.plugin -tags "$(BUILD_TAGS)" *.go

# Fail if any file is not gofmt-clean, listing the offenders.
fmt:
	@files=$$(gofmt -l .); \
	if [ -n "$$files" ]; then echo "gofmt needed:"; echo "$$files"; exit 1; fi

vet:
	go vet ./...

# `make test` runs everything: format check, vet, and the standards + unit tests.
test: fmt vet
	go test ./...

# Install the git pre-commit hook that runs `make test`.
hooks:
	@mkdir -p .git/hooks
	@ln -sf ../../scripts/pre-commit .git/hooks/pre-commit
	@echo "installed .git/hooks/pre-commit -> scripts/pre-commit"
