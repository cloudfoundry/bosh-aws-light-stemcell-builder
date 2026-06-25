package driver_test

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/url"

	"light-stemcell-builder/driver"
	"light-stemcell-builder/driver/manifests"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var s3Client *s3.Client

var _ = Describe("Machine Image Lifecycle", func() {
	BeforeEach(func() {
		s3Client = s3.NewFromConfig(creds.GetAwsConfig())
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
				imageURL, err := url.Parse(machineImage.GetURL)
				Expect(err).ToNot(HaveOccurred())

				params := &s3.HeadObjectInput{
					Bucket: aws.String(bucketName),
					Key:    aws.String(imageURL.Path),
				}
				headResp, err := s3Client.HeadObject(context.Background(), params)
				Expect(err).ToNot(HaveOccurred())

				Expect(string(headResp.ServerSideEncryption)).To(Equal("AES256"))
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
				imageURL, err := url.Parse(machineImage.GetURL)
				Expect(err).ToNot(HaveOccurred())

				params := &s3.HeadObjectInput{
					Bucket: aws.String(bucketName),
					Key:    aws.String(imageURL.Path),
				}
				headResp, err := s3Client.HeadObject(context.Background(), params)
				Expect(err).ToNot(HaveOccurred())

				Expect(string(headResp.ServerSideEncryption)).To(Equal("AES256"))

				imageURL, err = url.Parse(manifest.Parts.Part.HeadURL)
				Expect(err).ToNot(HaveOccurred())

				params = &s3.HeadObjectInput{
					Bucket: aws.String(bucketName),
					Key:    aws.String(imageURL.Path),
				}
				headResp, err = s3Client.HeadObject(context.Background(), params)
				Expect(err).ToNot(HaveOccurred())

				Expect(string(headResp.ServerSideEncryption)).To(Equal("AES256"))
			})
		})
	})
})

func checkUploadedUrl(getUrl string) int {
	parsedUrl, err := url.Parse(getUrl)
	Expect(err).ToNot(HaveOccurred())
	Expect([]string{"https", "s3"}).To(ContainElement(parsedUrl.Scheme))

	switch parsedUrl.Scheme {
	case "https":
		resp, err := http.Get(getUrl)
		Expect(err).ToNot(HaveOccurred())
		defer resp.Body.Close() //nolint:errcheck

		return http.StatusOK
	case "s3":
		_, err := s3Client.GetObject(context.Background(), &s3.GetObjectInput{
			Bucket: aws.String(parsedUrl.Host),
			Key:    aws.String(parsedUrl.Path),
		})

		if err == nil {
			return 200
		}
		var noSuchKey *s3types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return 404
		}
		Expect(err).ToNot(HaveOccurred())
	}

	return -1
}

func testMachineImageLifecycle(driverConfig resources.MachineImageDriverConfig, cb ...func(resources.MachineImage)) {
	createDriver := driver.NewCreateMachineImageDriver(GinkgoWriter, creds)

	machineImage, err := createDriver.Create(driverConfig)
	Expect(err).ToNot(HaveOccurred())

	statusCode := checkUploadedUrl(machineImage.GetURL)
	Expect(statusCode).To(Equal(http.StatusOK))

	if len(cb) > 0 {
		cb[0](machineImage)
	}

	deleteDriver := driver.NewDeleteMachineImageDriver(GinkgoWriter, creds)

	err = deleteDriver.Delete(machineImage)
	Expect(err).ToNot(HaveOccurred())

	statusCode = checkUploadedUrl(machineImage.GetURL)
	Expect(statusCode).To(Equal(http.StatusNotFound))
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
	defer resp.Body.Close() //nolint:errcheck

	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}
