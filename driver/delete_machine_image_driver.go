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

	d.logger.Printf("starting delete for the following resources: %s\n", strings.Join(machineImage.DeleteURLs, ", "))

	client := http.Client{
		Timeout: time.Duration(30 * time.Second),
	}

	for _, deleteURL := range machineImage.DeleteURLs {
		deleteReq, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
		if err != nil {
			return fmt.Errorf("Failed to create DELETE request for '%s': %s", deleteURL, err) //nolint:staticcheck
		}

		resp, err := client.Do(deleteReq)
		if err != nil {
			return fmt.Errorf("Failed to delete resource '%s': %s", deleteURL, err) //nolint:staticcheck
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			defer resp.Body.Close()                                                                                                   //nolint:errcheck
			respBody, _ := io.ReadAll(resp.Body)                                                                                      //nolint:errcheck
			return fmt.Errorf("Received invalid response code '%d' deleting resource '%s': %s", resp.StatusCode, deleteURL, respBody) //nolint:staticcheck
		}
	}

	return nil
}
