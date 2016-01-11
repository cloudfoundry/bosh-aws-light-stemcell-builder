#!/usr/bin/env bash

set -e

source builder-src/ci/tasks/utils.sh

check_param ami_description
check_param ami_virtualization_type
check_param ami_visibility
check_param us_ami_region
check_param us_ami_access_key
check_param us_ami_secret_key
check_param us_ami_bucket_name
check_param us_ami_destinations
check_param cn_ami_region
check_param cn_ami_access_key
check_param cn_ami_secret_key
check_param cn_ami_bucket_name

# export AWS_ACCESS_KEY_ID=$access_key
# export AWS_SECRET_ACCESS_KEY=$secret_key

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
ec2-describe-regions -O $us_ami_access_key -W $us_ami_secret_key --region $us_ami_region
# ec2-describe-regions -O $cn_ami_access_key -W $cn_ami_secret_key --region $cn_ami_region

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
  "ami_regions": [
    {
      "name":               "$us_ami_region",
      "credentials": {
        "access_key":       "$us_ami_access_key",
        "secret_key":       "$us_ami_secret_key"
      },
      "bucket_name":        "$us_ami_bucket_name",
      "destinations":       $us_ami_destinations
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
