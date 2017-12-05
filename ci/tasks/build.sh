#!/usr/bin/env bash

set -e -o pipefail

my_dir="$( cd $(dirname $0) && pwd )"
release_dir="$( cd ${my_dir} && cd ../.. && pwd )"

source ${release_dir}/ci/tasks/utils.sh

ami_kms_key_id=${ami_kms_key_id:-}
ami_server_side_encryption=${ami_server_side_encryption:-}

: ${bosh_io_bucket_name:?}
: ${ami_description:?}
: ${ami_virtualization_type:?}
: ${ami_visibility:?}
: ${ami_region:?}
: ${ami_access_key:?}
: ${ami_secret_key:?}
: ${ami_bucket_name:?}
: ${ami_encrypted:?}

export AWS_ACCESS_KEY_ID=$ami_access_key
export AWS_SECRET_ACCESS_KEY=$ami_secret_key
export AWS_DEFAULT_REGION=$ami_region

saved_ami_destinations="$( aws ec2 describe-regions \
  --query "Regions[?RegionName != '${ami_region}'][].RegionName" \
  | jq 'sort' -c )"

: ${ami_destinations:=$saved_ami_destinations}

stemcell_path=${PWD}/input-stemcell/*.tgz
output_path=${PWD}/light-stemcell/

echo "Checking if light stemcell already exists..."

original_stemcell_name="$(basename ${stemcell_path})"
light_stemcell_name="light-${original_stemcell_name}"

if [ "${ami_virtualization_type}" = "hvm" ]; then
  if [[ "${light_stemcell_name}" != *"-hvm"*  ]]; then
    light_stemcell_name="${light_stemcell_name/xen/xen-hvm}"
  fi
fi

bosh_io_light_stemcell_url="https://s3.amazonaws.com/$bosh_io_bucket_name/$light_stemcell_name"
set +e
wget --spider "$bosh_io_light_stemcell_url"
if [[ "$?" == "0" ]]; then
  echo "AWS light stemcell '$light_stemcell_name' already exists!"
  echo "You can download here: $bosh_io_light_stemcell_url"
  exit 1
fi
set -e

echo "Building light stemcell..."
echo "  Starting region: ${ami_region}"
echo "  Copy regions: ${ami_destinations}"

extracted_stemcell_dir=${PWD}/extracted-stemcell
mkdir -p ${extracted_stemcell_dir}
tar -C ${extracted_stemcell_dir} -xf ${stemcell_path}
tar -xf ${extracted_stemcell_dir}/image

# image format can be raw or stream optimized vmdk
stemcell_image="$(echo ${PWD}/root.*)"
stemcell_manifest=${extracted_stemcell_dir}/stemcell.MF
manifest_contents="$(cat ${stemcell_manifest})"

disk_regex="disk: ([0-9]+)"
format_regex="disk_format: ([a-z]+)"

[[ "${manifest_contents}" =~ ${disk_regex} ]]
disk_size_gb=$(mb_to_gb "${BASH_REMATCH[1]}")

[[ "${manifest_contents}" =~ ${format_regex} ]]
disk_format="${BASH_REMATCH[1]}"

stemcell_name=$(grep '^name: ' ${stemcell_manifest} | awk '{print $2}' | tr -d "\"'")
stemcell_version=$(grep '^version: ' ${stemcell_manifest} | awk '{print $2}' | tr -d "\"'")
ami_name="${stemcell_name}/${stemcell_version} from ${publisher_name:-unknown}"

config_path=${PWD}/config.json
cat > ${config_path} << EOF
{
  "ami_configuration": {
    "name":                 "$ami_name",
    "description":          "$ami_description",
    "virtualization_type":  "$ami_virtualization_type",
    "encrypted":            $ami_encrypted,
    "kms_key_id":           "$ami_kms_key_id",
    "visibility":           "$ami_visibility"
  },
  "ami_regions": [
    {
      "name":               "$ami_region",
      "credentials": {
        "access_key":       "$ami_access_key",
        "secret_key":       "$ami_secret_key"
      },
      "bucket_name":        "$ami_bucket_name",
      "server_side_encryption": "$ami_server_side_encryption",
      "destinations":       $ami_destinations
    }
  ]
}
EOF

pushd ${release_dir} > /dev/null
  . .envrc
  # Make sure we've closed the manifest file before writing to it
  go run src/light-stemcell-builder/main.go \
    -c ${config_path} \
    --image ${stemcell_image} \
    --format ${disk_format} \
    --volume-size ${disk_size_gb} \
    --manifest ${stemcell_manifest} \
    | tee tmp-manifest

  mv tmp-manifest ${stemcell_manifest}

popd

pushd ${extracted_stemcell_dir}
  > image
  # the bosh cli sees the stemcell as invalid if tar contents have leading ./
  tar -czf ${output_path}/${light_stemcell_name} *
popd
tar -tf ${output_path}/${light_stemcell_name}
