package driver

import (
	"light-stemcell-builder/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

func awsCreds(creds config.Credentials) *credentials.Credentials {
	if creds.AccessKey != "" && creds.SecretKey != "" {
		if creds.RoleArn != "" {
			return stscreds.NewCredentials(
				session.Must(session.NewSession(aws.NewConfig().WithCredentials(staticCredentials(creds)))),
				creds.RoleArn,
			)
		}

		return staticCredentials(creds)
	} else {
		return credentials.NewCredentials(&ec2rolecreds.EC2RoleProvider{
			Client: ec2metadata.New(session.Must(session.NewSession())),
		})
	}
}

func staticCredentials(creds config.Credentials) *credentials.Credentials {
	return credentials.NewStaticCredentialsFromCreds(
		credentials.Value{AccessKeyID: creds.AccessKey, SecretAccessKey: creds.SecretKey},
	)
}
