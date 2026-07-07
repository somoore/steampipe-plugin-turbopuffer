#!/usr/bin/env bash
# Extra pre-commit safety checks, on top of `make test`:
#   1. Dependencies pinned & hash-verified (go.sum integrity)
#   2. No secrets committed (trufflehog / gitleaks, whichever is installed)
#   3. No .env / credential files in the tree
#   4. Duplicative-code advisory (warn only)
#
# Blocking checks exit non-zero. The dedup check only warns.
# Run standalone with: make check-security
set -uo pipefail

cd "$(git rev-parse --show-toplevel)" || exit 1
fail=0

say()  { printf '\n\033[1m== %s\033[0m\n' "$1"; }
bad()  { printf '\033[31m  FAIL: %s\033[0m\n' "$1"; fail=1; }
ok()   { printf '\033[32m  ok: %s\033[0m\n' "$1"; }
warn() { printf '\033[33m  warn: %s\033[0m\n' "$1"; }

#### 1. Dependencies pinned & hash-verified ####################################
say "dependencies pinned and hash-verified"
# go.mod must not carry unpinned pseudo-selectors; every require needs a real
# version, and go.sum hashes must match the module cache.
if grep -Eq '=> ' go.mod; then
  bad "go.mod contains a replace directive — remove before release, or pin it deliberately"
else
  ok "no replace directives"
fi
if go mod verify >/dev/null 2>&1; then
  ok "go mod verify: all modules' hashes match go.sum"
else
  bad "go mod verify failed — a dependency's contents don't match its pinned hash"
fi
# Every require line must reference a semver tag (vX.Y.Z), not a bare branch.
unpinned=$(go list -m -f '{{if not .Main}}{{.Path}} {{.Version}}{{end}}' all 2>/dev/null \
  | awk '$2 !~ /^v[0-9]+\.[0-9]+\.[0-9]+/ {print $1" "$2}')
if [ -n "$unpinned" ]; then
  bad "unpinned/non-release dependency versions:"
  # shellcheck disable=SC2001  # sed indents multi-line output; clearer than expansion
  echo "$unpinned" | sed 's/^/    /'
else
  ok "all dependencies pinned to release versions"
fi

#### 2. No secrets committed ###################################################
say "secret scan"
if command -v trufflehog >/dev/null 2>&1; then
  if trufflehog filesystem . --no-update --only-verified --fail >/dev/null 2>&1; then
    ok "trufflehog: no verified secrets"
  else
    bad "trufflehog found verified secret(s) — run: trufflehog filesystem . --only-verified"
  fi
elif command -v gitleaks >/dev/null 2>&1; then
  if gitleaks detect --source . --no-git --no-banner >/dev/null 2>&1; then
    ok "gitleaks: no leaks"
  else
    bad "gitleaks found secret(s) — run: gitleaks detect --source ."
  fi
else
  warn "neither trufflehog nor gitleaks installed — skipping secret scan (brew install trufflehog)"
fi

#### 3. No .env / credential files #############################################
say "no credential files in tree"
# Track-worthy credential files that must never be committed.
env_hits=$(git ls-files -co --exclude-standard \
  | grep -E '(^|/)\.env($|\.)|(^|/).*\.spc$|(^|/)credentials(\.|$)|\.pem$|\.p12$|id_rsa' \
  | grep -vE '(^|/)config/turbopuffer\.spc$' || true)
if [ -n "$env_hits" ]; then
  bad "credential-shaped files present (add to .gitignore or remove):"
  # shellcheck disable=SC2001  # sed indents multi-line output; clearer than expansion
  echo "$env_hits" | sed 's/^/    /'
else
  ok "no .env / key / credential files"
fi
# Belt and suspenders: reject the turbopuffer key pattern anywhere in tracked text.
if git grep -nI 'tpuf_[A-Za-z0-9]\{20,\}' -- . >/dev/null 2>&1; then
  bad "a turbopuffer API key (tpuf_...) appears in the tree:"
  git grep -nI 'tpuf_[A-Za-z0-9]\{20,\}' -- . | sed 's/^/    /'
fi

#### 4. Duplicative code (advisory) ############################################
say "duplicative-code advisory"
# No heavyweight tool: flag identical normalized function bodies across the Go
# package. Near-dupes by design (the *Akas transforms) are expected; this is a
# nudge, not a gate.
dup=$(awk '
  /^func / {inbody=1; body=""; name=$0; next}
  inbody && /^}/ {
    gsub(/[ \t]+/,"",body)
    if (length(body) > 80) print body"\t"name
    inbody=0; next
  }
  inbody {body=body$0}
' turbopuffer/*.go 2>/dev/null | sort | awk -F'\t' '{c[$1]++} END{for(k in c) if(c[k]>1) print c[k]}' | wc -l | tr -d ' ')
if [ "${dup:-0}" -gt 0 ]; then
  warn "$dup exact-duplicate function body/bodies detected — consider extracting a helper (advisory only)"
else
  ok "no exact-duplicate function bodies"
fi

echo
if [ "$fail" -ne 0 ]; then
  printf '\033[31mSECURITY CHECKS FAILED\033[0m\n'
  exit 1
fi
printf '\033[32mall security checks passed\033[0m\n'
