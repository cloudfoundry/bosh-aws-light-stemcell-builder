package ec2

import (
	"fmt"
	"light-stemcell-builder/ec2/ec2instance"
	"time"
)

func RunInstance(aws AWS, instanceConfig ec2instance.Config) (ec2instance.Info, error) {
	instance, err := aws.RunInstance(instanceConfig)
	if err != nil {
		return ec2instance.Info{}, fmt.Errorf("Error running instances: %s", err)
	}

	waiterConfig := WaiterConfig{
		Resource:      instance,
		DesiredStatus: ec2instance.RunningStatus,
		PollTimeout:   10 * time.Minute,
	}

	_, err = WaitForStatus(aws.DescribeInstance, waiterConfig)
	if err != nil {
		return ec2instance.Info{}, fmt.Errorf("Error waiting for instance to be running: %s", err)
	}

	return instance, nil
}
