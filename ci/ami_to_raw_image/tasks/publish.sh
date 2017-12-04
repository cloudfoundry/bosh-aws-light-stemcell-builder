#!/usr/bin/env bash

set -e -o pipefail

my_dir="$( cd $(dirname $0) && pwd )"
release_dir="$( cd ${my_dir} && cd ../../.. && pwd )"
workspace_dir="$( cd ${release_dir} && cd .. && pwd )"

source ${release_dir}/ci/tasks/utils.sh

: ${ami_description:?}
: ${ami_virtualization_type:?}
: ${ami_visibility:?}
: ${ami_initial_region:?}
: ${ami_copy_regions:?}
: ${ami_access_key:?}
: ${ami_secret_key:?}
: ${ami_bucket_name:?}

# inputs
ami_path=${workspace_dir}/raw-machine-image/*.tgz

# outputs
output_path=${workspace_dir}/metadata

tmpdir="$(mktemp -d /tmp/publish-ami.XXXXX)"
trap '{ rm -rf ${tmpdir}; }' EXIT

extracted_ami_dir=${tmpdir}/extracted-ami

mkdir -p ${extracted_ami_dir}
pushd ${extracted_ami_dir} > /dev/null
  tar -xf ${ami_path}
  tar -xf ${extracted_ami_dir}/image
popd > /dev/null

# image format can be raw or stream optimized vmdk
ami_image="$(echo ${extracted_ami_dir}/root.*)"
ami_manifest=${extracted_ami_dir}/stemcell.MF
manifest_contents="$(cat ${ami_manifest})"

disk_regex="disk: ([0-9]+)"
format_regex="disk_format: ([a-z]+)"

[[ "${manifest_contents}" =~ ${disk_regex} ]]
disk_size_gb=$(mb_to_gb "${BASH_REMATCH[1]}")

[[ "${manifest_contents}" =~ ${format_regex} ]]
disk_format="${BASH_REMATCH[1]}"

config_path=${workspace_dir}/config.json
cat > ${config_path} << EOF
{
  "ami_configuration": {
    "description":          "$ami_description",
    "virtualization_type":  "$ami_virtualization_type",
    "visibility":           "$ami_visibility"
  },
  "ami_regions": [
    {
      "name":               "${ami_initial_region}",
      "credentials": {
        "access_key":       "$ami_access_key",
        "secret_key":       "$ami_secret_key"
      },
      "bucket_name":        "$ami_bucket_name",
      "destinations":       ${ami_copy_regions}
    }
  ]
}
EOF

echo "Configuration:"
cat $config_path

pushd ${release_dir} > /dev/null
  . .envrc
  # Make sure we've closed the manifest file before writing to it
  go run src/light-stemcell-builder/main.go \
    -c ${config_path} \
    --image ${ami_image} \
    --format ${disk_format} \
    --volume-size ${disk_size_gb} \
    --manifest ${ami_manifest} \
    | tee tmp-manifest

  mv tmp-manifest ${output_path}/stemcell.MF
  cat ${output_path}/stemcell.MF
popd
