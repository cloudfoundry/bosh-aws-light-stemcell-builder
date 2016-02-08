#!/usr/bin/env bash

set -e

release_dir="$( cd $(dirname $0)/../.. && pwd )"

source ${release_dir}/ci/tasks/utils.sh

tmp_dir="$(mktemp -d /tmp/stemcell_builder.XXXXXXX)"
trap '{ rm -rf ${tmpdir}; }' EXIT

: ${access_key:?}
: ${secret_key:?}
: ${bucket_name:?}
: ${region:?}
: ${ami_fixture_id:?}

export AWS_ACCESS_KEY_ID=$access_key
export AWS_SECRET_ACCESS_KEY=$secret_key

export AWS_BUCKET_NAME=$bucket_name
export AWS_REGION=$region
export AMI_FIXTURE_ID=$ami_fixture_id
export OUTPUT_STEMCELL_PATH=$PWD

echo "Checking Java configuration"
if hash java 2>/dev/null; then
  JAVA_EXEC="$(which java)"
else
  JAVA_EXEC="$JAVA_HOME/bin/java"
fi
${JAVA_EXEC} -version

echo "Checking EC2 CLI has been properly installed"

if ! hash ec2-describe-regions 2>/dev/null; then
  echo 'Error: Could not find "ec2-describe-regions" on PATH'
  exit 1
fi
ec2-describe-regions -O $access_key -W $secret_key --region $region

echo "Downloading machine image"
wget -O ${tmp_dir}/disk.iso http://tinycorelinux.net/6.x/x86_64/release/TinyCorePure64-6.4.1.iso
export LOCAL_DISK_IMAGE_PATH=${tmp_dir}/disk.iso

echo "Running integration tests"

pushd ${release_dir} > /dev/null
  . .envrc
  # TODO: re-enable errcheck (need to resolve errors found when `go get`ing)
  # go get github.com/kisielk/errcheck
  # errcheck light-stemcell-builder/...
  go test -v -timeout 1h30m light-stemcell-builder/...
popd
