#!/usr/bin/env bash

set -e

pushd light-stemcell-builder
    ginkgo -r --skipPackage driver,integration
popd
