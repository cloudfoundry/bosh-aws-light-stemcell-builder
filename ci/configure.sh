#!/usr/bin/env bash

set -e

FLY="${FLY_CLI:-fly}"

"$FLY" -t "${CONCOURSE_TARGET:-stemcells}" set-pipeline \
  -p bosh-aws-light-stemcell-builder \
  -c "$(dirname $0)/pipeline.yml"
