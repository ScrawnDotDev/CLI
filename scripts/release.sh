#!/usr/bin/env sh
set -eu

if [ "$#" -eq 0 ]; then
  set -- release --snapshot --clean
fi

if command -v goreleaser >/dev/null 2>&1; then
  exec goreleaser "$@"
fi

if command -v go >/dev/null 2>&1; then
  exec go run github.com/goreleaser/goreleaser/v2@latest "$@"
fi

echo "Neither 'goreleaser' nor 'go' was found. Install Go or Goreleaser, then try again." >&2
exit 1
