#!/bin/sh

set -eux

echo "Running gofmt..."
for TARGET in "${@}"; do
  find "${TARGET}" -name '*.go' ! -path '*/vendor/*' ! -path '*/.build/*' -exec gofmt -s -w {} \+
done
