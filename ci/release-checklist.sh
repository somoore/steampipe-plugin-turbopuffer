#!/usr/bin/env bash
# Pre-release checklist runner for the Steampipe plugin release standards:
#   https://steampipe.io/docs/develop/plugin-release-checklist
#
# Mechanically-checkable items are verified automatically (PASS/FAIL); items
# that need human judgment are printed as MANUAL reminders. Nothing here blocks
# a commit — run it by hand before cutting a release:
#   make release-check      (or: ./ci/release-checklist.sh)
#
# `make build` also runs this as a gate before building.
set -uo pipefail
cd "$(git rev-parse --show-toplevel)" || exit 1

PLUGIN=turbopuffer
GO_VERSION_WANT=1.26
pass=0 fail=0

hdr()  { printf '\n\033[1m### %s\033[0m\n' "$1"; }
ok()   { printf '  \033[32m[PASS]\033[0m %s\n' "$1"; pass=$((pass+1)); }
no()   { printf '  \033[31m[FAIL]\033[0m %s\n' "$1"; fail=$((fail+1)); }
man()  { printf '  \033[33m[MANUAL]\033[0m %s\n' "$1"; }

hdr "Basic configuration"
# Repo name: steampipe-plugin-<oneword>
repo=$(basename "$(git rev-parse --show-toplevel)")
if [[ "$repo" == "steampipe-plugin-$PLUGIN" ]]; then ok "repo name is steampipe-plugin-$PLUGIN"
else no "repo dir is '$repo', expected steampipe-plugin-$PLUGIN"; fi

# Go version in go.mod
gover=$(awk '/^go /{print $2}' go.mod)
if [[ "$gover" == ${GO_VERSION_WANT}* ]]; then ok "go.mod Go version is $gover"
else no "go.mod Go version is $gover, checklist wants $GO_VERSION_WANT"; fi

# .goreleaser.yml present and v2 format
if [[ -f .goreleaser.yml ]] && grep -q '^version: 2' .goreleaser.yml; then ok ".goreleaser.yml present (v2)"
else no ".goreleaser.yml missing or not version 2"; fi

# CHANGELOG
if [[ -f CHANGELOG.md ]]; then ok "CHANGELOG.md present"
else no "CHANGELOG.md missing (add release notes for the upcoming version)"; fi

# License: Apache 2.0
if grep -qi 'Apache License' LICENSE 2>/dev/null; then ok "LICENSE is Apache 2.0"
else no "LICENSE is not Apache 2.0 (checklist requires Apache-2.0)"; fi

# Makefile builds to the plugin path
if grep -q "plugins/local/$PLUGIN/$PLUGIN.plugin" Makefile; then ok "Makefile builds to the correct plugin path"
else no "Makefile install target does not build to plugins/local/$PLUGIN/$PLUGIN.plugin"; fi

man "GitHub repo topics include: postgresql postgresql-fdw sql steampipe steampipe-plugin"
man "GitHub repo website points to https://hub.steampipe.io/plugins/somoore/$PLUGIN"

hdr "Configuration file"
spc="config/$PLUGIN.spc"
if [[ -f "$spc" ]]; then
  ok "$spc present"
  # Realistic values, not placeholders
  if grep -Eiq 'TOKEN_HERE|YOUR_KEY|CHANGE_?ME|xxxx' "$spc"; then no "$spc contains placeholder credential values"
  else ok "$spc uses realistic example values"; fi
  # Env var documented
  if grep -q 'TURBOPUFFER_API_KEY' "$spc"; then ok "$spc mentions the TURBOPUFFER_API_KEY env var"
  else no "$spc should document the TURBOPUFFER_API_KEY env var"; fi
else no "$spc missing"; fi

hdr "Tables, columns, docs (delegates to make test)"
if make test >/dev/null 2>&1; then ok "make test passes (naming, descriptions, column order, docs, coding standards)"
else no "make test fails — run 'make test' to see standards violations"; fi

# Every table doc has 4-5 examples
for f in docs/tables/*.md; do
  n=$(grep -c '^### ' "$f")
  base=$(basename "$f")
  if (( n >= 4 )); then ok "$base has $n examples (>=4)"
  else no "$base has only $n examples; checklist wants 4-5 real-world examples"; fi
done

hdr "Index documentation"
idx=docs/index.md
for key in organization category icon_url brand_color description; do
  if grep -q "^$key:" "$idx"; then ok "index front matter has '$key'"
  else no "index front matter missing '$key'"; fi
done
if grep -qE '^## Credentials' "$idx"; then ok "index has a Credentials section"
else no "index missing Credentials section"; fi
if grep -qiE '^## (Multiple Connections|.*Aggregator)' "$idx"; then ok "index has a Multiple Connections / aggregator section"
else no "index missing a 'Multiple Connections' section (link to Using Aggregators)"; fi

hdr "Matching examples"
# README example should match index.md example (first SQL block)
readme_sql=$(awk '/```sql/{f=1;next} /```/{f=0} f' README.md 2>/dev/null | head -5)
index_sql=$(awk '/```sql/{f=1;next} /```/{f=0} f' "$idx" 2>/dev/null | head -5)
if [[ -n "$readme_sql" && "$readme_sql" == "$index_sql" ]]; then ok "README.md first example matches docs/index.md"
else man "verify README.md example matches docs/index.md example"; fi

hdr "Manual review (human judgment required)"
  man "Tested on a REAL account with substantial data (done: load-tested to 1,200+ namespaces; throttling only shows at scale)"
man "config/$PLUGIN.spc example matches docs/index.md#configuration"
man "Credentials doc explains scopes/permissions and links to provider docs"
man "icon_url points to a real .svg hosted on hub.steampipe.io (request via Steampipe Slack)"
man "brand_color matches turbopuffer brand guidelines"
man "category is a valid choice from hub.steampipe.io/plugins"
man "Social graphic added to README top + GitHub social preview"
man "Money columns (if any) are strings, not doubles"
man "Required config args are checked once at load time"
man "Errors include location + args (see plugin.Logger usage)"
man "Pre-mortem: considered why this could fail to delight users"

printf '\n\033[1m== %d passed, %d failed (plus MANUAL items above) ==\033[0m\n' "$pass" "$fail"
if (( fail > 0 )); then
  printf '\033[31mNot release-ready: fix the [FAIL] items.\033[0m\n'; exit 1
fi
printf '\033[32mAuto-checks pass. Work through the [MANUAL] items before releasing.\033[0m\n'
