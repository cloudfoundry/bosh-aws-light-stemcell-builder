#!/bin/bash

set -eux -o pipefail

my_dir="$( cd $(dirname $0) && pwd )"
release_dir="$( cd ${my_dir} && cd ../.. && pwd )"

input_stemcell_dir="${PWD}/input-stemcell"
output_stemcell_dir="${PWD}/converted-stemcell"

stemcell_path="${input_stemcell_dir}/*.tgz"
stemcell_name="$(basename ${stemcell_path})"

extracted_stemcell_dir=${PWD}/extracted-stemcell
mkdir -p ${extracted_stemcell_dir}
tar -xf ${stemcell_path} -C ${extracted_stemcell_dir}

sed -i'' -e 's/disk_format: raw/disk_format: vmdk/g' ${extracted_stemcell_dir}/stemcell.MF

pushd ${extracted_stemcell_dir}

  raw_stemcell_image=root.img
  optimized_stemcell_image=root.vmdk
  compressed_stemcell_image=image

  tar -xf ${compressed_stemcell_image}
  qemu-img convert -O vmdk -o subformat=streamOptimized ${raw_stemcell_image} ${optimized_stemcell_image}

  rm ${raw_stemcell_image}
  tar -cf ${compressed_stemcell_image} ${optimized_stemcell_image}
  rm ${optimized_stemcell_image}

  tar -cf ${output_stemcell_dir}/${stemcell_name} .
popd

tar -tf ${output_stemcell_dir}/${stemcell_name}
