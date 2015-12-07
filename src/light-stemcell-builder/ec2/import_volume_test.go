package ec2_test

import (
	"light-stemcell-builder/command"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2cli"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ec2", func() {
	ec2Credentials := ec2cli.Credentials{
		AccessKey: "",
		SecretKey: "",
	}

	getEC2Config := func() ec2cli.Config {
		conf := ec2cli.Config{
			BucketName: "bosh-demo-bucket",
			Region:     "cn-north-1",
			Credentials: &ec2cli.Credentials{
				AccessKey: ec2Credentials.AccessKey,
				SecretKey: ec2Credentials.SecretKey,
			},
		}
		return conf
	}

	ec2Config := getEC2Config()

	Describe("ImportVolume", func() {
		It("Creates an EBS volume from a local machine image and blocks until the volume is available", func() {
			imagePath := "/Users/pivotal/workspace/test_upload_instance_cn/tmp/root.img"
			Expect(imagePath).ToNot(BeEmpty())

			volID, err := ec2.ImportVolume(ec2Config, imagePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(volID).ToNot(BeEmpty())

			describeVolumes := exec.Command(
				"ec2-describe-volumes",
				"-O", ec2Config.AccessKey,
				"-W", ec2Config.SecretKey,
				"--region", ec2Config.Region,
				volID,
			)

			awk := exec.Command("awk", "{print $5}")
			cmds := []*exec.Cmd{describeVolumes, awk}

			out, err := command.RunPipeline(cmds)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal("available"))
		})
	})
})
