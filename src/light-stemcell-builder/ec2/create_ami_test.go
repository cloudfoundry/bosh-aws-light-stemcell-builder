package ec2_test

import (
	"light-stemcell-builder/command"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2ami"
	"light-stemcell-builder/ec2/ec2cli"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateAmi", func() {
	volumeID := "vol-ec1fe705"
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

	Describe("Configuration Validation", func() {
		It("checks that required fields have been set", func() {
			var c ec2ami.Config
			var err error

			c = ec2ami.Config{}

			err = c.Validate()
			Expect(err).To(MatchError("Description is required"))

			c = ec2ami.Config{
				Description: "some-description",
			}

			err = c.Validate()
			Expect(err).To(MatchError("VirtualizationType is required"))
		})
	})

	getImageField := func(amiID string, field int) (string, error) {
		describeImages := exec.Command(
			"ec2-describe-images",
			"-O", ec2Config.AccessKey,
			"-W", ec2Config.SecretKey,
			"--region", ec2Config.Region,
			amiID,
		)

		firstLine, err := command.NewSelectLine(1)
		Expect(err).ToNot(HaveOccurred())

		nthField, err := command.NewSelectField(field)
		Expect(err).ToNot(HaveOccurred())

		cmds := []*exec.Cmd{describeImages, firstLine, nthField}

		return command.RunPipeline(cmds)
	}

	It("creates an AMI from an EBS Volume", func() {
		amiConfig := ec2ami.Config{
			VirtualizationType: "hvm",
			Description:        "BOSH CI test AMI",
		}

		amiID, err := ec2.CreateAmi(volumeID, ec2Config, amiConfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(amiID).ToNot(BeEmpty())

		status, err := getImageField(amiID, 5)
		Expect(err).ToNot(HaveOccurred())
		Expect(status).To(Equal("available"))

		architecture, err := getImageField(amiID, 7)
		Expect(err).ToNot(HaveOccurred())
		Expect(architecture).To(Equal(ec2ami.AmiArchitecture))

		virtualiztionType, err := getImageField(amiID, 11)
		Expect(err).ToNot(HaveOccurred())
		Expect(virtualiztionType).To(Equal(amiConfig.VirtualizationType))
	})

	It("makes the AMI public if desired", func() {
		amiConfig := ec2ami.Config{
			Description:        "BOSH CI test AMI",
			Public:             true,
			VirtualizationType: "hvm",
		}

		amiID, err := ec2.CreateAmi(volumeID, ec2Config, amiConfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(amiID).ToNot(BeEmpty())

		status, err := getImageField(amiID, 5)
		Expect(err).ToNot(HaveOccurred())
		Expect(status).To(Equal("available"))

		accessibility, err := getImageField(amiID, 6)
		Expect(err).ToNot(HaveOccurred())
		Expect(accessibility).To(Equal(ec2ami.AmiPublicAccessibility))
	})
})
