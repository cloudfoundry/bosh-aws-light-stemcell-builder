package drivers_test

import (
	"light-stemcell-builder/config"
	"light-stemcell-builder/drivers"
	"light-stemcell-builder/resources"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MachineImageDriver", func() {
	It("uploads a machine image to S3 and creates a presigned URL", func() {
		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		Expect(accessKey).ToNot(BeEmpty(), "AWS_ACCESS_KEY_ID must be set")

		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		Expect(secretKey).ToNot(BeEmpty(), "AWS_SECRET_ACCESS_KEY must be set")

		region := os.Getenv("AWS_REGION")
		Expect(region).ToNot(BeEmpty(), "AWS_REGION must be set")

		creds := config.Credentials{
			AccessKey: accessKey,
			SecretKey: secretKey,
			Region:    region,
		}

		imagePath := os.Getenv("MACHINE_IMAGE_PATH")
		Expect(imagePath).ToNot(BeEmpty(), "MACHINE_IMAGE_PATH must be set")

		bucketName := os.Getenv("AWS_BUCKET_NAME")
		Expect(bucketName).ToNot(BeEmpty(), "AWS_BUCKET_NAME must be set")

		driverConfig := resources.MachineImageDriverConfig{
			MachineImagePath: imagePath,
			BucketName:       bucketName,
		}

		driver := drivers.NewMachineImageDriver(os.Stdout, creds)

		url, err := driver.Create(driverConfig)
		Expect(err).ToNot(HaveOccurred())

		resp, err := http.Get(url)
		Expect(err).ToNot(HaveOccurred())

		err = resp.Body.Close()
		Expect(err).ToNot(HaveOccurred())

		Expect(resp.StatusCode).To(Equal(200))

	})
})
