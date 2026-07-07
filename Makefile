STEAMPIPE_INSTALL_DIR ?= ~/.steampipe
BUILD_TAGS = netgo

.PHONY: install build test fmt vet hooks check-security release-check

install:
	go build -o $(STEAMPIPE_INSTALL_DIR)/plugins/local/turbopuffer/turbopuffer.plugin -tags "$(BUILD_TAGS)" *.go

# Release build: gate on the pre-release checklist, then build the binary.
# The checklist runner is gitignored; skip the gate gracefully if it's absent.
build: release-check install

# Run the pre-release checklist (auto-checks + manual reminders).
release-check:
	@if [ -x scripts/release-checklist.sh ]; then ./scripts/release-checklist.sh; \
	else echo "release-check: scripts/release-checklist.sh not present (gitignored) — skipping"; fi

# Fail if any file is not gofmt-clean, listing the offenders.
fmt:
	@files=$$(gofmt -l .); \
	if [ -n "$$files" ]; then echo "gofmt needed:"; echo "$$files"; exit 1; fi

vet:
	go vet ./...

# Dependency-pinning, secret, .env, and dedup checks (see ci/checks.sh).
check-security:
	@./ci/checks.sh

# `make test` runs everything: format check, vet, standards + unit tests, and
# the security checks.
test: fmt vet check-security
	go test ./...

# Point git at the versioned hooks dir so pre-commit AND pre-push run
# automatically. One setting, tracked in the repo, survives fresh clones
# (each clone just needs `make hooks` once).
hooks:
	@git config core.hooksPath ci/hooks
	@chmod +x ci/hooks/*
	@echo "git core.hooksPath -> ci/hooks (pre-commit + pre-push active)"
