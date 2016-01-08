#!/usr/bin/env bash

set -e

source builder-src/ci/tasks/utils.sh

check_param access_key
check_param secret_key
check_param bucket_name
check_param region
check_param copy_dests
check_param ami_description
check_param ami_is_public
check_param ami_virtualization_type

export AWS_ACCESS_KEY_ID=$access_key
export AWS_SECRET_ACCESS_KEY=$secret_key

echo "Setting environment variables"

export JAVA_HOME="/usr/lib/jvm/java-7-openjdk-amd64/jre"
echo "JAVA_HOME set to $JAVA_HOME"

export EC2_HOME="/usr/local/ec2/ec2-api-tools-1.7.5.1"
echo "EC2_HOME set to $EC2_HOME"

export PATH=$PATH:$EC2_HOME/bin

echo "Checking Java configuration"
$JAVA_HOME/bin/java -version

echo "Checking EC2 CLI has been properly installed"
which ec2-describe-regions
ec2-describe-regions -O $access_key -W $secret_key --region $region

stemcell_path=$(echo $PWD/input-stemcell/*.tgz)
output_path=$PWD/light-stemcell/

echo "Building light stemcell"

export CONFIG_PATH=$PWD/config.json

cat > $CONFIG_PATH << EOF
{
  "ami_configuration": {
    "description":          "$ami_description",
    "virtualization_type":  "$ami_virtualization_type",
    "visibility":           "$ami_visibility"
  },
  "regions": [
    {
      "name":               "$region",
      "credentials": {
        "access_key":       "$access_key",
        "secret_key":       "$secret_key"
      },
      "bucket_name":        "$bucket_name",
      "destinations":       $copy_dests
    }
  ]
}
EOF

echo "Configuration:"
cat $CONFIG_PATH

pushd builder-src > /dev/null
  . .envrc
  go run src/light-stemcell-builder/main.go -c $CONFIG_PATH -i $stemcell_path -o $output_path
popd
