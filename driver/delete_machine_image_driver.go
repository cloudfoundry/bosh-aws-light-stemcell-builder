package driver

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"light-stemcell-builder/config"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// The SDKDeleteMachineImageDriver deletes a previously uploaded machine image and manifest from S3
type SDKDeleteMachineImageDriver struct {
	s3Client *s3.S3
	logger   *log.Logger
}

// NewDeleteMachineImageDriver deletes a previously uploaded machine image and manifest from S3
func NewDeleteMachineImageDriver(logDest io.Writer, creds config.Credentials) *SDKDeleteMachineImageDriver {
	logger := log.New(logDest, "SDKDeleteMachineImageDriver ", log.LstdFlags)

	awsConfig := creds.GetAwsConfig().
		WithLogger(newDriverLogger(logger))

	s3Client := s3.New(session.Must(session.NewSession(awsConfig)))

	return &SDKDeleteMachineImageDriver{
		s3Client: s3Client,
		logger:   logger,
	}
}

// Delete will perform DELETE requests to all DeleteURLs on the machineImage
func (d *SDKDeleteMachineImageDriver) Delete(machineImage resources.MachineImage) error {
	deleteStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("completed Delete() in %f minutes\n", time.Since(deleteStartTime).Minutes())
	}(deleteStartTime)

	d.logger.Printf("starting delete for the following resources: %s\n", strings.Join(machineImage.DeleteURLs, ", "))

	client := http.Client{
		Timeout: time.Duration(30 * time.Second),
	}

	for _, deleteURL := range machineImage.DeleteURLs {
		deleteReq, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
		if err != nil {
			return fmt.Errorf("Failed to create DELETE request for '%s': %s", deleteURL, err)
		}

		resp, err := client.Do(deleteReq)
		if err != nil {
			return fmt.Errorf("Failed to delete resource '%s': %s", deleteURL, err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			defer resp.Body.Close()              //nolint:errcheck
			respBody, _ := io.ReadAll(resp.Body) // ignore ReadAll err, return http status code err instead
			return fmt.Errorf("Received invalid response code '%d' deleting resource '%s': %s", resp.StatusCode, deleteURL, respBody)
		}
	}

	return nil
}
