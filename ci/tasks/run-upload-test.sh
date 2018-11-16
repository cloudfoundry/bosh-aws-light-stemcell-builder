#!/usr/bin/env bash

set -ex

source bosh-cpi-src/ci/utils.sh
source director-state/director.env


pushd light-stemcell
  time bosh -n upload-stemcell *.tgz
popd

