package ec2

import (
	"fmt"
	"light-stemcell-builder/ec2/ec2instance"
	"time"
)

func TerminateInstance(aws AWS, instance ec2instance.Info) error {
	err := aws.TerminateInstance(instance)
	if err != nil {
		return fmt.Errorf("Error terminating instances: %s", err)
	}

	waiterConfig := WaiterConfig{
		Resource:      instance,
		DesiredStatus: ec2instance.TerminatedStatus,
		PollTimeout:   10 * time.Minute,
	}

	_, err = WaitForStatus(aws.DescribeInstance, waiterConfig)
	if err != nil {
		return fmt.Errorf("Error waiting for instance to be terminated: %s", err)
	}

	return nil
}
