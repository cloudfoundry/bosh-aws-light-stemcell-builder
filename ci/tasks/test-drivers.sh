#!/usr/bin/env bash
set -eu -o pipefail

REPO_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )/../.." && pwd )"
REPO_PARENT="$( cd "${REPO_ROOT}/.." && pwd )"

if [[ -n "${DEBUG:-}" ]]; then
  set -x
fi

# US Regions
export AWS_ACCOUNT=${aws_account_id?'must be set'}
export AWS_ACCESS_KEY_ID=${access_key?'must be set'}
export AWS_SECRET_ACCESS_KEY=${secret_key?'must be set'}
export AWS_BUCKET_NAME=${bucket_name?'must be set'}
export AWS_REGION=${region?'must be set'}
export AWS_DESTINATION_REGION=${copy_region?'must be set'}
export AWS_KMS_KEY_ID=${kms_key_id?'must be set'}
export MULTI_REGION_KEY=${kms_multi_region_key?'must be set'}
export MULTI_REGION_KEY_REPLICATION_TEST=${kms_multi_region_key_replication_test?'must be set'}

# Fixtures
export S3_MACHINE_IMAGE_URL=${uploaded_machine_image_url?'must be set'}
export S3_MACHINE_IMAGE_FORMAT=${uploaded_machine_image_format:="RAW"}
export EBS_VOLUME_ID=${existing_volume_id?'must be set'}
export EBS_SNAPSHOT_ID=${existing_snapshot_id?'must be set'}
export AMI_FIXTURE_ID=${ami_fixture_id?'must be set'}
export PRIVATE_AMI_FIXTURE_ID=${private_ami_fixture_id?'must be set'}

echo "Downloading machine image"
export MACHINE_IMAGE_PATH="${REPO_PARENT}/image.iso"
export MACHINE_IMAGE_FORMAT="RAW"
wget -O "${MACHINE_IMAGE_PATH}" http://tinycorelinux.net/7.x/x86_64/archive/7.1/TinyCorePure64-7.1.iso

echo "Running driver tests"

pushd "${REPO_ROOT}" > /dev/null
  # Run all driver specs in parallel to reduce test time
  spec_count="$(grep "It(" -r driver | wc -l)"
  go run github.com/onsi/ginkgo/v2/ginkgo -nodes "${spec_count}" -r driver
popd
