package driver

import (
	"light-stemcell-builder/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

func awsCreds(creds config.Credentials) *credentials.Credentials {
	if creds.AccessKey != "" && creds.SecretKey != "" {
		return credentials.NewStaticCredentials(creds.AccessKey, creds.SecretKey, "")
	} else {
		return credentials.NewCredentials(&ec2rolecreds.EC2RoleProvider{
			Client: ec2metadata.New(session.New(&aws.Config{})), //nolint:staticcheck
		})
	}
}
