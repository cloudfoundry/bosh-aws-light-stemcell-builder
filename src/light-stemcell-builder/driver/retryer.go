package driver

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/request"
)

func NewS3RetryerWithRetries(numRetries int) S3Retryer {
	//s3Retryer := S3Retryer{}
	//s3Retryer.NumMaxRetries = numRetries
	return S3Retryer{client.DefaultRetryer{NumMaxRetries: numRetries}}
}

// S3Retryer handles more error conditions than the default retryer when
// uploading chunks to S3
type S3Retryer struct {
	client.DefaultRetryer
}

// MaxRetries returns the configured number of NumMaxRetries, defaults to 3
func (r S3Retryer) MaxRetries() int {
	if r.NumMaxRetries <= 0 {
		return 3
	}
	return r.NumMaxRetries
}

// ShouldRetry returns a SerializationError if the response body was interrupted.
// S3Retryer will check for this error before invoking DefaultRetryer.ShouldRetry
func (r S3Retryer) ShouldRetry(req *request.Request) bool {
	if req.Error != nil {
		if err, ok := req.Error.(awserr.Error); ok {
			if err.Code() == "SerializationError" {
				return true
			}
		}
	}
	return r.DefaultRetryer.ShouldRetry(req)
}
