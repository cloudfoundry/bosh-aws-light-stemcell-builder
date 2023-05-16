package driver_test

import (
	"encoding/xml"
	"io"
	"net/http"
	"net/url"

	"light-stemcell-builder/driver"
	"light-stemcell-builder/driver/manifests"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Machine Image Lifecycle", func() {
	var s3Client *s3.S3

	BeforeEach(func() {
		awsSession, err := session.NewSession()
		Expect(err).ToNot(HaveOccurred())

		s3Client = s3.New(awsSession)
	})

	It("uploads a machine image to S3 with pre-signed URLs for GET and DELETE", func() {
		driverConfig := resources.MachineImageDriverConfig{
			MachineImagePath: machineImagePath,
			BucketName:       bucketName,
		}

		testMachineImageLifecycle(driverConfig)
	})

	Context("when ServerSideEncryption is specified", func() {
		It("uploads a machine image to S3 with pre-signed URLs for GET and DELETE", func() {
			driverConfig := resources.MachineImageDriverConfig{
				MachineImagePath:     machineImagePath,
				BucketName:           bucketName,
				ServerSideEncryption: "AES256",
			}

			testMachineImageLifecycle(driverConfig, func(machineImage resources.MachineImage) {
				imageURL, err := url.Parse(machineImage.GetURL) //nolint:ineffassign,staticcheck

				params := &s3.HeadObjectInput{
					Bucket: aws.String(bucketName),
					Key:    aws.String(imageURL.Path),
				}
				headResp, err := s3Client.HeadObject(params)
				Expect(err).ToNot(HaveOccurred())

				Expect(*headResp.ServerSideEncryption).To(Equal("AES256"))
			})
		})
	})

	It("uploads a machine image w/manifest to S3 with pre-signed URLs for GET and DELETE", func() {
		driverConfig := resources.MachineImageDriverConfig{
			MachineImagePath: machineImagePath,
			FileFormat:       machineImageFormat,
			BucketName:       bucketName,
			VolumeSizeGB:     3,
		}

		testMachineImageManifestLifecycle(driverConfig)
	})

	Context("when ServerSideEncryption is specified", func() {
		It("uploads a machine image to S3 with pre-signed URLs for GET and DELETE", func() {
			driverConfig := resources.MachineImageDriverConfig{
				MachineImagePath:     machineImagePath,
				FileFormat:           machineImageFormat,
				BucketName:           bucketName,
				VolumeSizeGB:         3,
				ServerSideEncryption: "AES256",
			}

			testMachineImageManifestLifecycle(driverConfig, func(machineImage resources.MachineImage, manifest manifests.ImportVolumeManifest) {
				imageURL, err := url.Parse(machineImage.GetURL) //nolint:ineffassign,staticcheck

				params := &s3.HeadObjectInput{
					Bucket: aws.String(bucketName),
					Key:    aws.String(imageURL.Path),
				}
				headResp, err := s3Client.HeadObject(params)
				Expect(err).ToNot(HaveOccurred())

				Expect(*headResp.ServerSideEncryption).To(Equal("AES256"))

				imageURL, err = url.Parse(manifest.Parts.Part.HeadURL) //nolint:ineffassign,staticcheck

				params = &s3.HeadObjectInput{
					Bucket: aws.String(bucketName),
					Key:    aws.String(imageURL.Path),
				}
				headResp, err = s3Client.HeadObject(params)
				Expect(err).ToNot(HaveOccurred())

				Expect(*headResp.ServerSideEncryption).To(Equal("AES256"))
			})
		})
	})
})

func testMachineImageLifecycle(driverConfig resources.MachineImageDriverConfig, cb ...func(resources.MachineImage)) {
	createDriver := driver.NewCreateMachineImageDriver(GinkgoWriter, creds)

	machineImage, err := createDriver.Create(driverConfig)
	Expect(err).ToNot(HaveOccurred())

	resp, err := http.Get(machineImage.GetURL)
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close()

	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	if len(cb) > 0 {
		cb[0](machineImage)
	}

	deleteDriver := driver.NewDeleteMachineImageDriver(GinkgoWriter, creds)

	err = deleteDriver.Delete(machineImage)
	Expect(err).ToNot(HaveOccurred())

	resp, err = http.Get(machineImage.GetURL)
	Expect(err).ToNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}

func testMachineImageManifestLifecycle(driverConfig resources.MachineImageDriverConfig, cb ...func(resources.MachineImage, manifests.ImportVolumeManifest)) {
	createDriver := driver.NewCreateMachineImageManifestDriver(GinkgoWriter, creds)

	machineImage, err := createDriver.Create(driverConfig)
	Expect(err).ToNot(HaveOccurred())

	resp, err := http.Get(machineImage.GetURL)
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close() //nolint:errcheck

	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	manifestBytes, err := io.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())

	m := manifests.ImportVolumeManifest{}
	err = xml.Unmarshal(manifestBytes, &m)
	Expect(err).ToNot(HaveOccurred())

	resp, err = http.Head(m.Parts.Part.HeadURL)
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close() //nolint:errcheck

	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(m.FileFormat).To(Equal(machineImageFormat))
	Expect(m.VolumeSizeGB).To(Equal(int64(3)))

	if len(cb) > 0 {
		cb[0](machineImage, m)
	}

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
}
