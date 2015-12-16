package ec2

import (
	"fmt"
	"time"
)

type StatusInfo interface {
	Status() string
}

type StatusResource interface {
	ID() string
}

type StatusFetcher func(StatusResource) (StatusInfo, error)

type WaiterConfig struct {
	Resource      StatusResource
	DesiredStatus string
	PollInterval  time.Duration
	PollTimeout   time.Duration
}

const (
	defaultPollTimeout  = 5 * time.Minute
	defaultPollInterval = 5 * time.Second
)

// TimeoutError is thrown when long polling times out
type TimeoutError struct {
	timeout  time.Duration
	resource StatusResource
}

func (e TimeoutError) Error() string {
	return fmt.Sprintf("timed out after %s polling on resource %s", e.timeout, e.resource.ID())
}

func WaitForStatus(status StatusFetcher, c WaiterConfig) (StatusInfo, error) {
	pollTimeout := c.PollTimeout
	if pollTimeout == 0 {
		pollTimeout = defaultPollTimeout
	}

	pollInterval := c.PollInterval
	if pollInterval == 0 {
		pollInterval = defaultPollInterval
	}

	fmt.Println(fmt.Sprintf("Waiting on %s to be desired status %s", c.Resource.ID(), c.DesiredStatus))
	timeout := time.After(pollTimeout)
	ticker := time.Tick(pollInterval)
	for {
		select {
		case <-timeout:
			fmt.Println(fmt.Sprintf("Timed out waiting for resource"))
			return nil, TimeoutError{timeout: pollTimeout, resource: c.Resource}

		case <-ticker:
			info, err := status(c.Resource)
			if err != nil {
				fmt.Println(fmt.Sprintf("Describe encountered error %s", err))
				return nil, err
			}
			if info.Status() == c.DesiredStatus {
				fmt.Println(fmt.Sprintf("%s matches desired status %s", c.Resource.ID(), c.DesiredStatus))
				return info, nil
			}
		}
	}
}
