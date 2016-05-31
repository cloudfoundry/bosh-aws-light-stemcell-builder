#!/bin/bash

set -eux -o pipefail

my_dir="$( cd $(dirname $0) && pwd )"
workspace_dir="$( cd ${my_dir} && cd ../../../.. && pwd )"

: ${AWS_ACCESS_KEY_ID:?}
: ${AWS_SECRET_ACCESS_KEY:?}
: ${SOURCE_AMI:?}
: ${REGION_NAME:=us-east-1}
: ${VAGRANT_AMI:=ami-fce3c696} # Ubuntu 14.04
: ${VM_USER:=ubuntu}
: ${SECURITY_GROUP_ID:?}
: ${PRIVATE_KEY_CONTENTS:?}
: ${PUBLIC_KEY_NAME:?}
: ${SUBNET_ID:?}

export AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID}
export AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}
export AWS_DEFAULT_REGION=${REGION_NAME}

# outputs
out_dir="${workspace_dir}/raw-machine-image"
mkdir -p ${out_dir}

# vagrant up
private_key_file=${workspace_dir}/vm_private_key.pem
echo "${PRIVATE_KEY_CONTENTS}" > ${private_key_file}
chmod go-r ${private_key_file}

export VAGRANT_CONFIG_FILE="${workspace_dir}/vagrant_light_stemcell_builder_config.json"
cat > "${VAGRANT_CONFIG_FILE}"<<EOF
{
  "AWS_ACCESS_KEY_ID": "${AWS_ACCESS_KEY_ID}",
  "AWS_SECRET_ACCESS_KEY": "${AWS_SECRET_ACCESS_KEY}",
  "AWS_DEFAULT_REGION": "${REGION_NAME}",
  "VM_AMI": "${VAGRANT_AMI}",
  "VM_NAME": "ami_to_raw_image",
  "VM_USER": "${VM_USER}",
  "AWS_SECURITY_GROUP": "${SECURITY_GROUP_ID}",
  "VM_KEYPAIR_NAME": "${PUBLIC_KEY_NAME}",
  "AWS_SUBNET_ID": "${SUBNET_ID}",
  "VM_PRIVATE_KEY_FILE": "${private_key_file}",
  "SOURCE_AMI": "${SOURCE_AMI}"
}
EOF

cleanup() {
  pushd ${my_dir} > /dev/null
    # vagrant destroy -f
  popd > /dev/null
}

trap cleanup EXIT

pushd ${my_dir} > /dev/null
  vagrant box add dummy https://github.com/mitchellh/vagrant-aws/raw/master/dummy.box --force
  vagrant up --provider=aws

  vagrant ssh-config > ./vagrant.ssh.config
  scp -F vagrant.ssh.config default:/vagrant/vmdk-ami.tgz ${out_dir}/
popd > /dev/null
