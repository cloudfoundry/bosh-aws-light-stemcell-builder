package ec2_test

import (
	"bufio"
	"bytes"
	"fmt"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2ami"
	"light-stemcell-builder/uuid"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CopyAmi Lifecycle", func() {
	aws := getAWSImplmentation()

	amiExistsWithName := func(creds *ec2.Credentials, region string, amiName string) bool {
		describeImage := exec.Command(
			"ec2-describe-images",
			"-O", creds.AccessKey,
			"-W", creds.SecretKey,
			"--region", region,
			"--filter", fmt.Sprintf("name=%s", amiName),
		)

		stderr := &bytes.Buffer{}
		describeImage.Stderr = stderr

		stdout, err := describeImage.Output()
		if err != nil {
			Fail(fmt.Sprintf("getting image information: %s, stderr: %s", err, stderr.String()))
		}

		outputLines := []string{}
		scanner := bufio.NewScanner(bytes.NewReader(stdout))
		for scanner.Scan() {
			outputLines = append(outputLines, scanner.Text())
		}

		if len(outputLines) == 0 {
			return false
		}

		return true
	}

	It("copies the AMI to multiple regions", func() {
		Expect(amiFixtureRegion).ToNot(Equal("cn-north-1"), "an AMI cannot be copied succesfully between the China region and any other region")

		testName, err := uuid.New("bosh-ci-test")
		amiConfig := ec2ami.Config{AmiID: amiFixtureID, Region: amiFixtureRegion, Description: "Copy AMI Lifecycle test AMI", VirtualizationType: "paravirtual"}
		amiConfig.UniqueName = testName

		amiInfo := ec2ami.Info{InputConfig: amiConfig}
		destinations := []string{"us-west-1", "us-west-2"}
		amiCollection, err := ec2.CopyAmis(aws, amiInfo, destinations)
		Expect(err).ToNot(HaveOccurred())

		for _, destination := range destinations {
			ami := amiCollection.Get(destination)
			Expect(ami.AmiID).ToNot(BeEmpty())
			Expect(ami.Status()).To(Equal(ec2.VolumeAvailableStatus))
			Expect(ami.Architecture).To(Equal(ec2ami.AmiArchitecture))
			Expect(ami.VirtualizationType).To(Equal(amiConfig.VirtualizationType))
			Expect(ami.Accessibility).To(Equal(ec2ami.AmiPublicAccessibility))
		}

		err = ec2.DeleteCopiedAmis(aws, amiCollection)
		Expect(err).ToNot(HaveOccurred())
		for _, destination := range destinations {
			Expect(amiExistsWithName(aws.GetConfig().Credentials, destination, testName)).To(BeFalse())
		}
	})

	It("cleans up all AMIs if copying to a region fails", func() {
		testName, err := uuid.New("bosh-ci-test")
		Expect(err).ToNot(HaveOccurred())

		amiConfig := ec2ami.Config{AmiID: amiFixtureID, Region: amiFixtureRegion, Description: "Copy AMI Lifecycle test AMI", VirtualizationType: "paravirtual"}
		amiConfig.UniqueName = testName

		amiInfo := ec2ami.Info{InputConfig: amiConfig}
		destinations := []string{"us-west-1", "us-west-2", "cn-north-1"}
		_, err = ec2.CopyAmis(aws, amiInfo, destinations)
		Expect(err).To(HaveOccurred())

		Expect(amiExistsWithName(aws.GetConfig().Credentials, "us-west-1", testName)).To(BeFalse())
		Expect(amiExistsWithName(aws.GetConfig().Credentials, "us-west-2", testName)).To(BeFalse())
	})
})
