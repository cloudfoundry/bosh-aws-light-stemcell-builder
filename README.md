## Light Stemcell Builder for AWS

This tool takes a raw machine image and a configuration file and creates a collection of AMIs.
Any AWS region including China is supported.

#### AWS Setup for Publishing

1. Create an S3 bucket for intermediate artifacts (e.g. `light-stemcells-for-project-XXX`)
1. Create an AWS IAM policy based on the JSON contained in `builder-policy.json`
1. Replace the bucket placeholder in your policy with the bucket created in step 1
    ```diff
      "Resource": [
    -    "arn:aws:s3:::<disk-image-file-bucket>",
    -    "arn:aws:s3:::<disk-image-file-bucket>/*"
    +    "arn:aws:s3:::light-stemcells-for-project-XXX",
    +    "arn:aws:s3:::light-stemcells-for-project-XXX/*"
      ]
    ```
    Note: The arn for AWS GovCloud region is `aws-us-gov`. It looks like this: `"arn:aws-us-gov:s3:::<disk-image-file-bucket>"`
1. Create an AWS IAM user and attach the policy created in steps 2, 3.
1. Create the `vmimport` AWS role as detailed [here](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/VMImportPrerequisites.html#iam-permissions-image), specifying the previously created bucket in place of `<disk-image-file-bucket>`; see [example IAM policy](iam-policy.json).
1. Replicate these steps in a separate AWS China account if publishing to China.

#### IAM User Setup for Integration Testing

1. Follow steps in "AWS Setup for Publishing"
1. Create an IAM policy based on the JSON contained in `integration-test-policy.json`
1. Attach the policy you created in step 2 to the existing publishing user

#### Testing

Unit testing:
```
ginkgo -r --skipPackage driver,integration
```

#### Example Usage

Example config:
```
{
  "ami_configuration": {
    "description":          "Your description here",
    "virtualization_type":  "hvm",
    "visibility":           "public"
  },
  "ami_regions": [
    {
      "name":               "us-east-1",
      "credentials": {
        "access_key":       "US_ACCESS_KEY_ID",
        "secret_key":       "US_ACCESS_SECRET_KEY"
      },
      "bucket_name":        "US_BUCKET_NAME",
      "destinations":       ["us-west-1", "us-west-2"]
    },
    {
      "name":               "cn-north-1",
      "credentials": {
        "access_key":       "CN_ACCESS_KEY_ID",
        "secret_key":       "CN_ACCESS_SECRET_KEY"
      },
      "bucket_name":        "CN_BUCKET_NAME"
    }
  ]
}
```

Usage:
```
./light-stemcell-builder -c config.json --image root.img --manifest stemcell.MF > updated-stemcell.MF
```

Example Output:
```
name: bosh-aws-xen-hvm-ubuntu-trusty-go_agent
version: "3202"
bosh_protocol: "1"
sha1: f0c10bb5e8b7fee9c29db15bbb4ae481e398eab6
operating_system: ubuntu-trusty
cloud_properties:
  ami:
    cn-north-1: ami-69ae6504
    us-east-1: ami-e62f158c
    us-west-1: ami-947e0df4
    us-west-2: ami-54328238
```

#### Troubleshooting

If the `vmimport` role is not present, you will receive this error from the light stemcell builder:

> Error publishing AMIs to us-east-1: creating snapshot: creating import snapshot task: InvalidParameter: The sevice role <vmimport> does not exist or does not have sufficient permissions for the service to continue
	status code: 400, request id:
