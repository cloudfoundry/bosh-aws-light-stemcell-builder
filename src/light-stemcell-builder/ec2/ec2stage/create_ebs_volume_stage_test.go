package ec2stage_test

import (
	"errors"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2stage"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"log"
	"io/ioutil"
)

var _ = Describe("CreateEbsVolume", func() {
	noopCleaner := func(aws ec2.AWS, taskID string) error {
		return nil
	}

	logger := log.New(ioutil.Discard, "", log.LstdFlags)

	Describe("Run()", func() {
		noopUndoer := func(aws ec2.AWS, volumeID string) error {
			return nil
		}

		It("passes the image path to the runner", func() {
			var called bool
			var path string

			dummyRunner := func(aws ec2.AWS, imagePath string) (ec2.ConversionTaskInfo, error) {
				called = true
				path = imagePath
				return ec2.ConversionTaskInfo{}, nil
			}

			s := ec2stage.NewCreateEBSVolumeStage(dummyRunner, noopCleaner, noopUndoer, dummyAWS{})
			_, err := s.Run(logger, "/tmp/some-image-path")
			Expect(err).ToNot(HaveOccurred())
			Expect(called).To(BeTrue())
			Expect(path).To(Equal("/tmp/some-image-path"))
		})

		It("returns errors from the runner", func() {
			errorRunner := func(aws ec2.AWS, imagePath string) (ec2.ConversionTaskInfo, error) {
				return ec2.ConversionTaskInfo{}, errors.New("this is an error")
			}

			s := ec2stage.NewCreateEBSVolumeStage(errorRunner, noopCleaner, noopUndoer, dummyAWS{})
			_, err := s.Run(logger, "/tmp/some-image-path")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("this is an error"))
		})

		It("expects input to be a string", func() {
			dummyRunner := func(aws ec2.AWS, imagePath string) (ec2.ConversionTaskInfo, error) {
				return ec2.ConversionTaskInfo{}, nil
			}

			s := ec2stage.NewCreateEBSVolumeStage(dummyRunner, noopCleaner, noopUndoer, dummyAWS{})
			_, err := s.Run(logger, 42)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected type string, got: int"))
		})

		It("returns a string as data", func() {
			dummyRunner := func(aws ec2.AWS, imagePath string) (ec2.ConversionTaskInfo, error) {
				return ec2.ConversionTaskInfo{EBSVolumeID: "dummy-ebs-volume-id"}, nil
			}

			var dummyString string
			s := ec2stage.NewCreateEBSVolumeStage(dummyRunner, noopCleaner, noopUndoer, dummyAWS{})
			rawData, err := s.Run(logger, "/tmp/some-image-path")
			Expect(err).ToNot(HaveOccurred())
			Expect(rawData).To(BeAssignableToTypeOf(dummyString))
			Expect(rawData.(string)).To(Equal("dummy-ebs-volume-id"))
		})

		It("uses the cleaner to clean sometime during Run()", func() {
			var called bool
			dummyRunner := func(aws ec2.AWS, imagePath string) (ec2.ConversionTaskInfo, error) {
				return ec2.ConversionTaskInfo{}, nil
			}
			dummyCleaner := func(aws ec2.AWS, taskID string) error {
				called = true
				return nil
			}

			s := ec2stage.NewCreateEBSVolumeStage(dummyRunner, dummyCleaner, noopUndoer, dummyAWS{})
			_, err := s.Run(logger, "/tmp/some-image-path")
			Expect(err).ToNot(HaveOccurred())
			Expect(called).To(BeTrue())
		})
	})
	Describe("Rollback()", func() {
		It("passes the volume ID to the undoer", func() {
			var called bool
			var volID string

			dummyRunner := func(aws ec2.AWS, imagePath string) (ec2.ConversionTaskInfo, error) {
				return ec2.ConversionTaskInfo{EBSVolumeID: "dummy-ebs-volume-id"}, nil
			}

			dummyUndoer := func(aws ec2.AWS, volumeID string) error {
				called = true
				volID = volumeID
				return nil
			}
			s := ec2stage.NewCreateEBSVolumeStage(dummyRunner, noopCleaner, dummyUndoer, dummyAWS{})
			_, err := s.Run(logger, "/tmp/some-image-path")
			Expect(err).ToNot(HaveOccurred())
			err = s.Rollback(logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(called).To(BeTrue())
			Expect(volID).To(Equal("dummy-ebs-volume-id"))
		})
	})
})
