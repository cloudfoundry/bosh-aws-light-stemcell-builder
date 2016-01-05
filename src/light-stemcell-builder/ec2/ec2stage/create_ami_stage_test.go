package ec2stage_test

import (
	"errors"
	"io/ioutil"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2ami"
	"light-stemcell-builder/ec2/ec2stage"
	"light-stemcell-builder/ec2/fakes"
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateAmi", func() {
	logger := log.New(ioutil.Discard, "", log.LstdFlags)

	Describe("Run()", func() {
		noopUndoer := func(aws ec2.AWS, amiInfo ec2ami.Info) error {
			return nil
		}

		It("passes the volume ID to the runner", func() {
			var called bool
			var volID string

			dummyRunner := func(aws ec2.AWS, volumeID string, amiConfig ec2ami.Config) (ec2ami.Info, error) {
				called = true
				volID = volumeID
				return ec2ami.Info{AmiID: "dummy-ami-id"}, nil
			}

			s := ec2stage.NewCreateAmiStage(dummyRunner, noopUndoer, &fakes.FakeAWS{}, ec2ami.Config{})
			_, err := s.Run(logger, "dummy-ebs-volume-id")
			Expect(err).ToNot(HaveOccurred())
			Expect(called).To(BeTrue())
			Expect(volID).To(Equal("dummy-ebs-volume-id"))
		})

		It("returns errors from the runner", func() {
			errorRunner := func(aws ec2.AWS, volumeID string, amiConfig ec2ami.Config) (ec2ami.Info, error) {
				return ec2ami.Info{}, errors.New("this is an error")
			}

			s := ec2stage.NewCreateAmiStage(errorRunner, noopUndoer, &fakes.FakeAWS{}, ec2ami.Config{})
			_, err := s.Run(logger, "dummy-ebs-volume-id")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("this is an error"))
		})

		It("expects input to be a string", func() {
			dummyRunner := func(aws ec2.AWS, volumeID string, amiConfig ec2ami.Config) (ec2ami.Info, error) {
				return ec2ami.Info{AmiID: "dummy-ami-id"}, nil
			}

			s := ec2stage.NewCreateAmiStage(dummyRunner, noopUndoer, &fakes.FakeAWS{}, ec2ami.Config{})
			_, err := s.Run(logger, 42)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected type string, got: int"))
		})

		It("returns an ec2ami.Info as data", func() {
			dummyAmi := ec2ami.Info{AmiID: "dummy-ami-id"}
			dummyRunner := func(aws ec2.AWS, volumeID string, amiConfig ec2ami.Config) (ec2ami.Info, error) {
				return dummyAmi, nil
			}

			s := ec2stage.NewCreateAmiStage(dummyRunner, noopUndoer, &fakes.FakeAWS{}, ec2ami.Config{})
			rawData, err := s.Run(logger, "dummy-ebs-volume-id")
			Expect(err).ToNot(HaveOccurred())
			Expect(rawData).To(BeAssignableToTypeOf(dummyAmi))
			Expect(rawData.(ec2ami.Info)).To(Equal(dummyAmi))
		})
	})

	Describe("Rollback()", func() {
		It("passes the AMI Info to the undoer", func() {
			var called bool
			var info ec2ami.Info

			dummyAmiInfo := ec2ami.Info{AmiID: "dummy-ami-id"}
			dummyRunner := func(aws ec2.AWS, volumeID string, amiConfig ec2ami.Config) (ec2ami.Info, error) {
				return dummyAmiInfo, nil
			}

			dummyUndoer := func(aws ec2.AWS, amiInfo ec2ami.Info) error {
				called = true
				info = amiInfo
				return nil
			}
			s := ec2stage.NewCreateAmiStage(dummyRunner, dummyUndoer, &fakes.FakeAWS{}, ec2ami.Config{})
			_, err := s.Run(logger, "dummy-ebs-volume-id")
			Expect(err).ToNot(HaveOccurred())
			err = s.Rollback(logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(called).To(BeTrue())
			Expect(info).To(Equal(dummyAmiInfo))
		})
	})
})
