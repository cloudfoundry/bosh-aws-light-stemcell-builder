package driver

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
)

// NewS3RetryerWithRetries returns an S3Retryer configured with the specified number of max retries.
func NewS3RetryerWithRetries(numRetries int) S3Retryer {
	return S3Retryer{NumMaxRetries: numRetries}
}

// S3Retryer handles more error conditions than the default retryer when
// uploading chunks to S3. It retries on serialization errors in addition to
// the standard retry conditions.
type S3Retryer struct {
	NumMaxRetries int
}

// MaxRetries returns the configured number of NumMaxRetries, defaults to 3.
func (r S3Retryer) MaxRetries() int {
	if r.NumMaxRetries <= 0 {
		return 3
	}
	return r.NumMaxRetries
}

// IsErrorRetryable returns true if the error should trigger a retry.
// It checks for serialization errors (e.g. broken S3 connections) in
// addition to any standard retry conditions.
func (r S3Retryer) IsErrorRetryable(err error) bool {
	if err != nil && strings.Contains(err.Error(), "SerializationError") {
		return true
	}
	return false
}

// AsAWSRetryer returns an aws.Retryer configured with this S3Retryer's settings,
// suitable for passing to aws.Config.Retryer.
func (r S3Retryer) AsAWSRetryer() aws.Retryer {
	maxAttempts := r.MaxRetries()
	serializationChecker := retry.IsErrorRetryableFunc(func(err error) aws.Ternary {
		if err != nil && strings.Contains(err.Error(), "SerializationError") {
			return aws.TrueTernary
		}
		return aws.UnknownTernary
	})
	return retry.NewStandard(func(o *retry.StandardOptions) {
		o.MaxAttempts = maxAttempts
		o.Retryables = append([]retry.IsErrorRetryable{serializationChecker}, o.Retryables...)
	})
}
