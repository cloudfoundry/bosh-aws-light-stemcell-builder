package driver

import (
	"light-stemcell-builder/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

func awsCreds(creds config.Credentials, logger aws.Logger) *credentials.Credentials {
	return credentials.NewChainCredentials(
		[]credentials.Provider{
			&ec2rolecreds.EC2RoleProvider{
				Client: ec2metadata.New(session.New(aws.NewConfig().WithLogger(logger))),
			},
			&credentials.StaticProvider{Value: credentials.Value{
				AccessKeyID:     creds.AccessKey,
				SecretAccessKey: creds.SecretKey,
			}},
		})
}
