package ec2cli

// Config specifies credentials to connect to AWS and what bucket to use
type Config struct {
	BucketName string
	Region     string
	*Credentials
}

type Credentials struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}
