package ec2ami_test

import (
	"encoding/json"
	"light-stemcell-builder/ec2/ec2ami"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ami", func() {
	Describe("Configuration Validation", func() {
		It("checks that required fields have been set", func() {
			var c ec2ami.Config
			var err error

			c = ec2ami.Config{}

			err = c.Validate()
			Expect(err).To(MatchError("Region is required"))

			c = ec2ami.Config{
				Region: "some-region",
			}

			err = c.Validate()
			Expect(err).To(MatchError("Description is required"))

			c = ec2ami.Config{
				Region:      "some-region",
				Description: "some-description",
			}

			err = c.Validate()
			Expect(err).To(MatchError("VirtualizationType is required"))

			c = ec2ami.Config{
				Region:             "some-region",
				Description:        "some-description",
				VirtualizationType: "some-virtualization-type",
			}

			err = c.Validate()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("JSON encoding", func() {
		config := ec2ami.Config{
			Description:        "Some Description",
			Public:             true,
			VirtualizationType: "hvm",
			UniqueName:         "Some Unique Name",
			Region:             "us-east-1",
			AmiID:              "ami-id",
		}
		info := ec2ami.Info{
			InputConfig:        config,
			AmiID:              "ami-id",
			Region:             "us-east-1",
			SnapshotID:         "snap-id",
			Accessibility:      "public",
			Name:               "Some Unique Name",
			ImageStatus:        "available",
			KernelId:           "aki-id",
			Architecture:       "x86_64",
			VirtualizationType: "hvm",
			StorageType:        "ebs",
		}
		It("outputs only the expected fields", func() {
			b, err := json.Marshal(info)
			Expect(err).ToNot(HaveOccurred())
			Expect(b).To(MatchJSON(`{
				"ami_id": "ami-id",
				"region": "us-east-1",
				"snapshot_id": "snap-id",
				"name": "Some Unique Name",
				"virtualization_type": "hvm",
				"accessibility": "public"
			}`))
		})
		It("successfully encodes an AMI map", func() {
			collection := ec2ami.NewCollection()
			collection.Add("us-east-1a", info)
			collection.Add("us-east-1b", info)
			collection.Add("us-east-1c", info)
			amiMap := collection.GetAll()
			b, err := json.Marshal(amiMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(b).To(MatchJSON(`{
				"us-east-1a": {
					"ami_id": "ami-id",
					"region": "us-east-1",
					"snapshot_id": "snap-id",
					"name": "Some Unique Name",
					"virtualization_type": "hvm",
					"accessibility": "public"
				},
				"us-east-1b": {
					"ami_id": "ami-id",
					"region": "us-east-1",
					"snapshot_id": "snap-id",
					"name": "Some Unique Name",
					"virtualization_type": "hvm",
					"accessibility": "public"
				},
				"us-east-1c": {
					"ami_id": "ami-id",
					"region": "us-east-1",
					"snapshot_id": "snap-id",
					"name": "Some Unique Name",
					"virtualization_type": "hvm",
					"accessibility": "public"
				}
			}`))
		})
	})
})
