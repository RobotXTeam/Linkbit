#!/usr/bin/env sh
set -eu

missing=0

if command -v go >/dev/null 2>&1; then
  go version
elif [ -x ".tools/go/bin/go" ]; then
  .tools/go/bin/go version
else
  echo "missing: go 1.23+"
  missing=1
fi

if ! command -v node >/dev/null 2>&1; then
  echo "missing: node 20+"
  missing=1
else
  node --version
fi

if ! command -v npm >/dev/null 2>&1; then
  echo "missing: npm"
  missing=1
else
  npm --version
fi

exit "$missing"
