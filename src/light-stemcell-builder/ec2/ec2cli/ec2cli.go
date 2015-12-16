package ec2cli

import (
	"light-stemcell-builder/ec2"
)

type EC2Cli struct {
	config ec2.Config
}

func (e *EC2Cli) Configure(c ec2.Config) {
	e.config = c
}

func (e *EC2Cli) GetConfig() ec2.Config {
	return e.config
}
