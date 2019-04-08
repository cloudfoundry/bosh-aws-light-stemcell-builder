#!/usr/bin/env bash

set -e

release_dir="$( cd $(dirname $0)/../.. && pwd )"

echo "Running unit tests"

pushd ${release_dir} > /dev/null
  . .envrc
  # TODO: re-enable errcheck (need to resolve errors found when `go get`ing)
  # go get github.com/kisielk/errcheck
  # errcheck light-stemcell-builder/...

  ginkgo -p -r -skipPackage "driver,integration" src/light-stemcell-builder/
  ginkgo -p -r src/light-stemcell-builder/driverset # driverset is skipped by previous command
popd
