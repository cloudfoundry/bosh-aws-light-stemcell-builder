package driver_test

import (
	"light-stemcell-builder/driver"
	"net/http"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Retryer", func() {
	It("returns a default for the number of max retries if not specified", func() {
		r := driver.S3Retryer{}
		Expect(r.MaxRetries()).To(Equal(3))
	})
	It("returns the number of max retries", func() {
		r := driver.S3Retryer{}
		r.NumMaxRetries = 10
		Expect(r.MaxRetries()).To(Equal(10))
	})
	It("should retry upon serialization error on the response", func() {
		r := driver.S3Retryer{}
		req := &request.Request{}
		req.HTTPResponse = &http.Response{StatusCode: 200}
		req.Error = awserr.New("SerializationError", "failed to decode S3 XML error response", nil)
		Expect(r.ShouldRetry(req)).To(BeTrue())
	})
})
