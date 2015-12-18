package ec2_test

import (
	"light-stemcell-builder/ec2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"light-stemcell-builder/ec2/ec2cli"
	"testing"
	"os"
)

// this needs to be in a standard (non-china AWS region)
var amiFixtureID = os.Getenv("AMI_FIXTURE_ID")
var amiFixtureRegion = os.Getenv("AWS_REGION")
var localDiskImagePath = os.Getenv("LOCAL_DISK_IMAGE_PATH")

func getAWSImplmentation() ec2.AWS {
	credentials := ec2.Credentials{
		AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
	}
	config := ec2.Config{
		BucketName:  os.Getenv("AWS_BUCKET_NAME"),
		Region:      os.Getenv("AWS_REGION"),
		Credentials: &credentials,
	}

	impl := &ec2cli.EC2Cli{}
	impl.Configure(config)
	return impl
}

func TestEc2(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ec2 Suite")
}
