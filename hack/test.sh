#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

TIMEOUT=${TIMEOUT:--timeout 120s}

ROOT_DIR=enn-policy
TEST_POLICY="enn-policy/pkg/policy"
TEST_IPSET="enn-policy/pkg/util/ipset"
TEST_IPTABLES="enn-policy/pkg/util/iptables"

find_dirs() {
  (
    find . -name "*test.go" -print0 | xargs -0n1 dirname | grep -v "./vendor" | sed "s|^\./|${ROOT_DIR}/|" | LC_ALL=C sort -u
  )
}

testcases=$(find_dirs)
for testcase in ${testcases}; do
  if [ "$testcase" = "$TEST_POLICY" ]; then
    echo "test case need kernel so skip case" $testcase
  elif [ "$testcase" = "$TEST_IPSET" ]; then
    echo "test case need kernel so skip case" $testcase
  elif [ "$testcase" = "$TEST_IPTABLES" ]; then
    echo "test case need kernel so skip case" $testcase
  else
    go test ${testcase} ${TIMEOUT}
  fi
done