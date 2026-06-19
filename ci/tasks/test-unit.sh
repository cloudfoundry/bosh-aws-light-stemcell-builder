#!/usr/bin/env bash
set -eu -o pipefail

REPO_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )/../.." && pwd )"
REPO_PARENT="$( cd "${REPO_ROOT}/.." && pwd )"

if [[ -n "${DEBUG:-}" ]]; then
  set -x
fi

echo "Running unit tests"

pushd "${REPO_ROOT}" > /dev/null
  go run github.com/onsi/ginkgo/v2/ginkgo -p -r --skip-package "driver,integration"
  go run github.com/onsi/ginkgo/v2/ginkgo -p -r driverset # driverset is skipped by previous command
popd
