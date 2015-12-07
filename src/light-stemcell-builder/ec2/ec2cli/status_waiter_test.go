package ec2cli_test

import (
	"errors"
	"light-stemcell-builder/ec2/ec2cli"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StatusWaiter", func() {
	config := ec2cli.WaiterConfig{
		ResourceID:    "some-resource-id",
		DesiredStatus: "desired",
		PollInterval:  time.Millisecond,
		PollTimeout:   2 * time.Millisecond,
	}

	Context("when the status fetcher returns an error", func() {
		It("returns the error", func() {
			errorFetcher := func(c ec2cli.Config, resourceID string) (string, error) {
				return "", errors.New("this returns an error")
			}

			err := ec2cli.WaitForStatus(errorFetcher, config)
			Expect(err).To(MatchError("this returns an error"))
		})
	})

	Context("when the status fetcher returns the desired status", func() {
		It("returns no error", func() {
			statusFetcher := func(c ec2cli.Config, resourceID string) (string, error) {
				return "desired", nil
			}

			err := ec2cli.WaitForStatus(statusFetcher, config)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when waiting times out", func() {
		It("returns a timeout error", func() {
			statusFetcher := func(c ec2cli.Config, resourceID string) (string, error) {
				return "not-desired", nil
			}

			err := ec2cli.WaitForStatus(statusFetcher, config)
			Expect(err).To(MatchError("timed out after 2ms polling on resource some-resource-id"))
		})
	})

})
