#!/usr/bin/env bash
set -eu -o pipefail

REPO_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )/../.." && pwd )"
REPO_PARENT="$( cd "${REPO_ROOT}/.." && pwd )"

if [[ -n "${DEBUG:-}" ]]; then
  set -x
fi

# US Regions
export AWS_ACCESS_KEY_ID=${access_key?'must be set'}
export AWS_SECRET_ACCESS_KEY=${secret_key?'must be set'}
export AWS_BUCKET_NAME=${bucket_name?'must be set'}
export AWS_REGION=${region?'must be set'}
export AWS_DESTINATION_REGION=${copy_region?'must be set'}

# # China Region
# export AWS_CN_ACCESS_KEY_ID=${cn_access_key?'must be set'}
# export AWS_CN_SECRET_ACCESS_KEY=${cn_secret_key?'must be set'}
# export AWS_CN_BUCKET_NAME=${cn_bucket_name?'must be set'}
# export AWS_CN_REGION=${cn_region?'must be set'}

echo "Downloading machine image"
export MACHINE_IMAGE_PATH="${REPO_PARENT}/image.iso"
export MACHINE_IMAGE_FORMAT="RAW"
wget -O "${MACHINE_IMAGE_PATH}" http://tinycorelinux.net/7.x/x86_64/archive/7.1/TinyCorePure64-7.1.iso

echo "Running integration tests"

pushd "${REPO_ROOT}" > /dev/null
  go run github.com/onsi/ginkgo/v2/ginkgo -v -r integration
popd
