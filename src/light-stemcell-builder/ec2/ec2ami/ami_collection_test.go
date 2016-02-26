package ec2ami_test

import (
	"light-stemcell-builder/ec2/ec2ami"

	"gopkg.in/yaml.v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func randomAMIInfo(region string) ec2ami.Info {
	return ec2ami.Info{
		AmiID:              "ami-" + region,
		Region:             region,
		SnapshotID:         "random snapshot",
		Accessibility:      "public",
		Name:               "random name",
		ImageStatus:        "available",
		KernelId:           "aki-something",
		Architecture:       "x86_64",
		VirtualizationType: "hvm",
		StorageType:        "EBS",
		InputConfig: ec2ami.Config{
			Description:        "Blah",
			Public:             true,
			VirtualizationType: "hvm",
			UniqueName:         "random name",
			Region:             region,
			AmiID:              "ami-" + region,
		},
	}
}

var _ = Describe("AmiCollection", func() {
	Describe("MarshalYAML", func() {
		It("allows the AMI collection to be marshaled to YAML", func() {
			amiCollection := ec2ami.NewCollection()
			regions := []string{"us-west-1", "eu-west-1", "sa-east-1", "us-west-2",
				"eu-central-1", "ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "us-east-1"}
			for _, region := range regions {
				amiCollection.Add(region, randomAMIInfo(region))
			}

			output, err := yaml.Marshal(amiCollection)
			Expect(err).ToNot(HaveOccurred())

			for _, region := range regions {
				Expect(output).To(MatchRegexp("(?m)^" + region + ": ami-" + region + "$"))
			}
		})
	})
})
