package test_helpers

import (
	"light-stemcell-builder/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

func AwsConfigFrom(configCredentials config.Credentials) *aws.Config {
	var awsCredentials *credentials.Credentials

	if configCredentials.AccessKey != "" && configCredentials.SecretKey != "" {
		awsCredentials = credentials.NewStaticCredentialsFromCreds(
			credentials.Value{AccessKeyID: configCredentials.AccessKey, SecretAccessKey: configCredentials.SecretKey},
		)

		if configCredentials.RoleArn != "" {
			staticConfig := aws.NewConfig().WithRegion(configCredentials.Region).WithCredentials(awsCredentials)
			awsCredentials = stscreds.NewCredentials(
				session.Must(session.NewSession(staticConfig)),
				configCredentials.RoleArn,
			)
		}
	} else {
		awsCredentials = credentials.NewCredentials(&ec2rolecreds.EC2RoleProvider{
			Client: ec2metadata.New(session.Must(session.NewSession())),
		})
	}

	return aws.NewConfig().WithRegion(configCredentials.Region).WithCredentials(awsCredentials)
}
