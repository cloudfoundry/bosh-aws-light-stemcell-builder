package drivers_test

import (
	"encoding/xml"
	"io"
	"io/ioutil"
	"light-stemcell-builder/config"
	"light-stemcell-builder/drivers/manifests"
	"light-stemcell-builder/driversets"
	"light-stemcell-builder/resources"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MachineImageManifestDriver", func() {
	It("uploads a machine image to S3 and creates a presigned URL for an import volume manifest", func() {
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

		ds := driversets.NewIsolatedRegionDriverSet(GinkgoWriter, creds)
		driver := ds.CreateMachineImageDriver()

		machineImage, err := driver.Create(driverConfig)
		Expect(err).ToNot(HaveOccurred())

		resp, err := http.Get(machineImage.GetURL)
		Expect(err).ToNot(HaveOccurred())

		defer func(reader io.ReadCloser) {
			err = reader.Close()
			Expect(err).ToNot(HaveOccurred())
		}(resp.Body)

		Expect(resp.StatusCode).To(Equal(200))

		manifestBytes, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		m := manifests.ImportVolumeManifest{}
		err = xml.Unmarshal(manifestBytes, &m)
		Expect(err).ToNot(HaveOccurred())
	})
})
