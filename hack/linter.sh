#!/bin/bash

set -e -o pipefail

if [ "$DISABLE_LINTER" == "true" ]
then
  exit 0
fi

docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v1.42.1 golangci-lint run --verbose --deadline 15m0s
