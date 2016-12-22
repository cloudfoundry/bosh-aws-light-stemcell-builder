#!/usr/bin/env bash

set -e

: ${AWS_DEFAULT_REGION:?}
: ${AWS_ACCESS_KEY_ID:?}
: ${AWS_SECRET_ACCESS_KEY:?}

builder_src=builder-src/ci/config/aws-regions.json

aws ec2 describe-regions \
  --query "Regions[?RegionName != '${AWS_DEFAULT_REGION}'][].RegionName" \
  | jq 'sort' -c > $builder_src

cp -r builder-src/* updated-builder-src/
