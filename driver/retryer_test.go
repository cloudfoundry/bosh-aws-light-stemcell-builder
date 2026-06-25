package driver_test

import (
	"errors"

	"light-stemcell-builder/driver"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Retryer", func() {
	Describe("NewS3RetryerWithRetries", func() {
		It("sets the number of max retries", func() {
			r := driver.NewS3RetryerWithRetries(33)
			Expect(r.MaxRetries()).To(Equal(33))
		})
	})

	Describe("S3Retryer", func() {
		It("returns a default for the number of max retries if not specified", func() {
			r := driver.S3Retryer{}
			Expect(r.MaxRetries()).To(Equal(3))
		})

		It("returns the number of max retries", func() {
			r := driver.S3Retryer{}
			r.NumMaxRetries = 10
			Expect(r.MaxRetries()).To(Equal(10))
		})

		It("should retry upon serialization error", func() {
			r := driver.S3Retryer{}
			err := errors.New("SerializationError: failed to decode S3 XML error response")
			Expect(r.IsErrorRetryable(err)).To(BeTrue())
		})

		It("should not retry on non-serialization errors", func() {
			r := driver.S3Retryer{}
			err := errors.New("some other error")
			// Delegates to Standard which returns false for plain errors
			Expect(r.IsErrorRetryable(err)).To(BeFalse())
		})
	})
})
