package ec2cli

import (
	"fmt"
	"time"
)

type statusInfo interface {
	Status() string
}

type StatusFetcher func(Config, string) (statusInfo, error)

type WaiterConfig struct {
	ResourceID    string
	DesiredStatus string
	PollInterval  time.Duration
	PollTimeout   time.Duration
	FetcherConfig Config
}

const (
	defaultPollTimeout  = 5 * time.Minute
	defaultPollInterval = 5 * time.Second
)

const (
	imageAvailableStatus    = "available"
	snapshotCompletedStatus = "completed"
)

// TimeoutError is thrown when long polling times out
type TimeoutError struct {
	timeout    time.Duration
	resourceID string
}

func (e TimeoutError) Error() string {
	return fmt.Sprintf("timed out after %s polling on resource %s", e.timeout, e.resourceID)
}

func WaitForStatus(status StatusFetcher, c WaiterConfig) error {
	pollTimeout := c.PollTimeout
	if pollTimeout == 0 {
		pollTimeout = defaultPollTimeout
	}

	pollInterval := c.PollInterval
	if pollInterval == 0 {
		pollInterval = defaultPollInterval
	}

	timeout := time.After(pollTimeout)
	ticker := time.Tick(pollInterval)
	for {
		select {
		case <-timeout:
			return TimeoutError{timeout: pollTimeout, resourceID: c.ResourceID}

		case <-ticker:
			info, err := status(c.FetcherConfig, c.ResourceID)
			if err != nil {
				return err
			}

			if info.Status() == c.DesiredStatus {
				return nil
			}
		}
	}
}
