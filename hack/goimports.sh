#!/bin/sh

set -eux
GOFLAGS='' go install golang.org/x/tools/cmd/goimports@v0.13.0
echo "Running goimports..."
for TARGET in "${@}"; do
  find "${TARGET}" -name '*.go' ! -path '*/vendor/*' ! -path '*/.build/*' ! -path '*/zz_generated*' -exec goimports -w {} \+
done
