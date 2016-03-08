package driver_test

import (
	"encoding/xml"
	"io/ioutil"
	"light-stemcell-builder/config"
	"light-stemcell-builder/driver"
	"light-stemcell-builder/driver/manifests"
	"light-stemcell-builder/resources"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Machine Image Lifecycle", func() {

	var (
		creds       config.Credentials
		imagePath   string
		imageFormat string
		bucketName  string
	)

	BeforeEach(func() {
		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		Expect(accessKey).ToNot(BeEmpty(), "AWS_ACCESS_KEY_ID must be set")

		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		Expect(secretKey).ToNot(BeEmpty(), "AWS_SECRET_ACCESS_KEY must be set")

		region := os.Getenv("AWS_REGION")
		Expect(region).ToNot(BeEmpty(), "AWS_REGION must be set")

		creds = config.Credentials{
			AccessKey: accessKey,
			SecretKey: secretKey,
			Region:    region,
		}

		imagePath = os.Getenv("MACHINE_IMAGE_PATH")
		Expect(imagePath).ToNot(BeEmpty(), "MACHINE_IMAGE_PATH must be set")

		imageFormat = os.Getenv("MACHINE_IMAGE_FORMAT")
		Expect(imageFormat).ToNot(BeEmpty(), "MACHINE_IMAGE_FORMAT must be set")

		bucketName = os.Getenv("AWS_BUCKET_NAME")
		Expect(bucketName).ToNot(BeEmpty(), "AWS_BUCKET_NAME must be set")
	})

	It("uploads a machine image to S3 with pre-signed URLs for GET and DELETE", func() {
		driverConfig := resources.MachineImageDriverConfig{
			MachineImagePath: imagePath,
			FileFormat:       imageFormat,
			BucketName:       bucketName,
		}

		createDriver := driver.NewCreateMachineImageDriver(GinkgoWriter, creds)

		machineImage, err := createDriver.Create(driverConfig)
		Expect(err).ToNot(HaveOccurred())

		resp, err := http.Get(machineImage.GetURL)
		Expect(err).ToNot(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		deleteDriver := driver.NewDeleteMachineImageDriver(GinkgoWriter, creds)

		err = deleteDriver.Delete(machineImage)
		Expect(err).ToNot(HaveOccurred())

		resp, err = http.Get(machineImage.GetURL)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	})

	It("uploads a machine image w/manifest to S3 with pre-signed URLs for GET and DELETE", func() {
		driverConfig := resources.MachineImageDriverConfig{
			MachineImagePath: imagePath,
			FileFormat:       imageFormat,
			BucketName:       bucketName,
		}

		createDriver := driver.NewCreateMachineImageManifestDriver(GinkgoWriter, creds)

		machineImage, err := createDriver.Create(driverConfig)
		Expect(err).ToNot(HaveOccurred())

		resp, err := http.Get(machineImage.GetURL)
		Expect(err).ToNot(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		manifestBytes, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		m := manifests.ImportVolumeManifest{}
		err = xml.Unmarshal(manifestBytes, &m)
		Expect(err).ToNot(HaveOccurred())

		resp, err = http.Head(m.Parts.Part.HeadURL)
		Expect(err).ToNot(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		deleteDriver := driver.NewDeleteMachineImageDriver(GinkgoWriter, creds)

		err = deleteDriver.Delete(machineImage)
		Expect(err).ToNot(HaveOccurred())

		resp, err = http.Get(machineImage.GetURL)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

		resp, err = http.Head(m.Parts.Part.HeadURL)
		Expect(err).ToNot(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	})
})
