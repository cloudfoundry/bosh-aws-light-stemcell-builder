#!/usr/bin/env bash

set -e

release_dir="$( cd $(dirname $0)/../.. && pwd )"

source ${release_dir}/ci/tasks/utils.sh

tmp_dir="$(mktemp -d /tmp/stemcell_builder.XXXXXXX)"
trap '{ rm -rf ${tmpdir}; }' EXIT

: ${access_key:?must be set}
: ${secret_key:?must be set}
: ${bucket_name:?must be set}
: ${region:?must be set}
: ${copy_region:?must be set}
: ${cn_access_key:?must be set}
: ${cn_secret_key:?must be set}
: ${cn_bucket_name:?must be set}
: ${cn_region:?must be set}
: ${ami_fixture_id:?must be set}
: ${existing_volume_id:?must be set}
: ${existing_snapshot_id:?must be set}
: ${uploaded_machine_image_url:?must be set}

# US Regions
export AWS_ACCESS_KEY_ID=$access_key
export AWS_SECRET_ACCESS_KEY=$secret_key
export AWS_BUCKET_NAME=$bucket_name
export AWS_REGION=$region
export AWS_DESTINATION_REGION=${copy_region}

# China Region
export AWS_CN_ACCESS_KEY_ID=$cn_access_key
export AWS_CN_SECRET_ACCESS_KEY=$cn_secret_key
export AWS_CN_BUCKET_NAME=$cn_bucket_name
export AWS_CN_REGION=$cn_region

# Fixtures
export S3_MACHINE_IMAGE_URL=${uploaded_machine_image_url}
export EBS_VOLUME_ID=${existing_volume_id}
export EBS_SNAPSHOT_ID=${existing_snapshot_id}
export AMI_FIXTURE_ID=${ami_fixture_id}

echo "Downloading machine image"
export MACHINE_IMAGE_PATH=${tmp_dir}/image.iso
wget -O ${MACHINE_IMAGE_PATH} http://tinycorelinux.net/7.x/x86_64/release/TinyCorePure64-7.0.iso

echo "Running all tests"

pushd ${release_dir} > /dev/null
  . .envrc
  # TODO: re-enable errcheck (need to resolve errors found when `go get`ing)
  # go get github.com/kisielk/errcheck
  # errcheck light-stemcell-builder/...

  ginkgo -p -r src/light-stemcell-builder/
popd
