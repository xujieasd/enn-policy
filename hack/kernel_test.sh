#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

TIMEOUT=${TIMEOUT:--timeout 120s}

ROOT_DIR=enn-policy

find_dirs() {
  (
    find . -name "*test.go" -print0 | xargs -0n1 dirname | grep -v "./vendor" | sed "s|^\./|${ROOT_DIR}/|" | LC_ALL=C sort -u
  )
}

testcases=$(find_dirs)
for testcase in ${testcases}; do
  go test ${testcase} ${TIMEOUT}
done