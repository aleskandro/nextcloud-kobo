#!/bin/sh

set -eux

#cd /tmp
GOFLAGS='' go install github.com/securego/gosec/v2/cmd/gosec@v2.19.0
gosec -severity medium -confidence medium "${@}"
#cd -
