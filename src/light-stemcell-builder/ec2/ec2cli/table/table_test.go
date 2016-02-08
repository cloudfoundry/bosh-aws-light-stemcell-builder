package table_test

import (
	"light-stemcell-builder/ec2/ec2cli/table"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EC2 CLI Table", func() {

	const rawInput = "INSTANCE\ti-1ae5555\tami-60b7777\nPUBLICIPADDRESS\t52.52.52.52"
	type Instance struct {
		InstanceID string `key:"INSTANCE" position:"0"`
		AmiID      string `key:"INSTANCE" position:"1"`
		PublicIP   string `key:"PUBLICIPADDRESS" position:"0"`
	}

	It("returns the value for a given key and position", func() {
		instance := Instance{}
		err := table.Marshall(rawInput, &instance)
		Expect(err).ToNot(HaveOccurred())

		Expect(instance.InstanceID).To(Equal("i-1ae5555"))
		Expect(instance.AmiID).To(Equal("ami-60b7777"))
		Expect(instance.PublicIP).To(Equal("52.52.52.52"))
	})

	It("returns an error if target is not a pointer", func() {
		instance := Instance{}
		err := table.Marshall(rawInput, instance)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Expected marshall target to be pointer"))
	})

	It("returns an error if input is not tab separated", func() {
		badInput := "INSTANCE i-1ae5555 ami-60b7777\nPUBLICIPADDRESS 52.52.52.52"
		instance := Instance{}
		err := table.Marshall(badInput, &instance)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Expected fields to be tab separated in input"))
	})

	It("returns an error if `key` is not specified in struct tag", func() {
		type badInstance struct {
			InstanceID string `position:"0"`
		}
		instance := badInstance{}
		err := table.Marshall(rawInput, &instance)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Expected to find `key` in struct tag"))
	})

	It("returns an error if `position` is not specified in struct tag", func() {
		type badInstance struct {
			InstanceID string `key:"INSTANCE"`
		}
		instance := badInstance{}
		err := table.Marshall(rawInput, &instance)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Expected to find `position` in struct tag"))
	})

	It("does not error when the input provides extra values", func() {
		type partialInstance struct {
			InstanceID string `key:"INSTANCE" position:"0"`
		}
		instance := partialInstance{}
		err := table.Marshall(rawInput, &instance)
		Expect(err).ToNot(HaveOccurred())
	})

	It("does not error when the input has empty values", func() {
		partialInput := "INSTANCE\ti-1ae5555\t\nPUBLICIPADDRESS\t52.52.52.52"
		instance := Instance{}
		err := table.Marshall(partialInput, &instance)
		Expect(err).ToNot(HaveOccurred())
		Expect(instance.InstanceID).To(Equal("i-1ae5555"))
		Expect(instance.AmiID).To(Equal(""))
		Expect(instance.PublicIP).To(Equal("52.52.52.52"))
	})

	It("returns an error if `position` is out of range", func() {
		type badInstance struct {
			InstanceID string `key:"INSTANCE" position:"99"`
		}
		instance := badInstance{}
		err := table.Marshall(rawInput, &instance)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Position `99` is out of range for fields"))
	})
})
