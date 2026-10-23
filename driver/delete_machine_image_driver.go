package driver

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"light-stemcell-builder/config"
	"light-stemcell-builder/resources"
)

// The SDKDeleteMachineImageDriver deletes a previously uploaded machine image and manifest from S3
type SDKDeleteMachineImageDriver struct {
	logger *log.Logger
}

// NewDeleteMachineImageDriver deletes a previously uploaded machine image and manifest from S3
func NewDeleteMachineImageDriver(logDest io.Writer, _ config.Credentials) *SDKDeleteMachineImageDriver {
	logger := log.New(logDest, "SDKDeleteMachineImageDriver ", log.LstdFlags)

	return &SDKDeleteMachineImageDriver{
		logger: logger,
	}
}

// Delete will perform DELETE requests to all DeleteURLs on the machineImage
func (d *SDKDeleteMachineImageDriver) Delete(machineImage resources.MachineImage) error {
	deleteStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("completed Delete() in %f minutes\n", time.Since(deleteStartTime).Minutes())
	}(deleteStartTime)

	d.logger.Printf("starting delete for %d resources\n", len(machineImage.DeleteURLs))

	client := http.Client{
		Timeout: time.Duration(30 * time.Second),
	}

	for _, deleteURL := range machineImage.DeleteURLs {
		deleteReq, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
		if err != nil {
			return fmt.Errorf("invalid DELETE request for %s", urlForLog(deleteURL)) //nolint:staticcheck
		}

		resp, err := client.Do(deleteReq)
		if err != nil {
			return fmt.Errorf("failed to delete resource %s: %s", urlForLog(deleteURL), requestError(err)) //nolint:staticcheck
		}
		resp.Body.Close() //nolint:errcheck
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("received response code %d deleting resource %s", resp.StatusCode, urlForLog(deleteURL))
		}
	}

	return nil
}
