package driver_test

import (
	"encoding/xml"
	"io/ioutil"
	"light-stemcell-builder/config"
	"light-stemcell-builder/driver"
	"light-stemcell-builder/driver/manifests"
	"light-stemcell-builder/resources"
	"net/http"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/aws/awserr"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Machine Image Lifecycle", func() {

	var (
		s3Client                          *s3.S3
		creds                             config.Credentials
		imagePath                         string
		imageFormat                       string
		bucketName                        string
		testMachineImageLifecycle         func(resources.MachineImageDriverConfig, ...func(resources.MachineImage))
		testMachineImageManifestLifecycle func(resources.MachineImageDriverConfig, ...func(resources.MachineImage, manifests.ImportVolumeManifest))
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

		s3Client = s3.New(session.New())

		imagePath = os.Getenv("MACHINE_IMAGE_PATH")
		Expect(imagePath).ToNot(BeEmpty(), "MACHINE_IMAGE_PATH must be set")

		imageFormat = os.Getenv("MACHINE_IMAGE_FORMAT")
		Expect(imageFormat).ToNot(BeEmpty(), "MACHINE_IMAGE_FORMAT must be set")

		bucketName = os.Getenv("AWS_BUCKET_NAME")
		Expect(bucketName).ToNot(BeEmpty(), "AWS_BUCKET_NAME must be set")
	})

	It("uploads a machine image to S3 with s3 URLs for GET and DELETE", func() {
		driverConfig := resources.MachineImageDriverConfig{
			MachineImagePath: imagePath,
			BucketName:       bucketName,
		}

		testMachineImageLifecycle(driverConfig)
	})

	Context("when ServerSideEncryption is specified", func() {
		It("uploads a machine image to S3 with s3 URLs for GET and DELETE", func() {
			driverConfig := resources.MachineImageDriverConfig{
				MachineImagePath:     imagePath,
				BucketName:           bucketName,
				ServerSideEncryption: "AES256",
			}

			testMachineImageLifecycle(driverConfig, func(machineImage resources.MachineImage) {
				imageURL, err := url.Parse(machineImage.GetURL)

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

	It("uploads a machine image w/manifest to S3 with s3 URLs for GET and DELETE", func() {
		driverConfig := resources.MachineImageDriverConfig{
			MachineImagePath: imagePath,
			FileFormat:       imageFormat,
			BucketName:       bucketName,
			VolumeSizeGB:     3,
		}

		testMachineImageManifestLifecycle(driverConfig)
	})

	Context("when ServerSideEncryption is specified", func() {
		It("uploads a machine image to S3 with s3 URLs for GET and DELETE", func() {
			driverConfig := resources.MachineImageDriverConfig{
				MachineImagePath:     imagePath,
				FileFormat:           imageFormat,
				BucketName:           bucketName,
				VolumeSizeGB:         3,
				ServerSideEncryption: "AES256",
			}

			testMachineImageManifestLifecycle(driverConfig, func(machineImage resources.MachineImage, manifest manifests.ImportVolumeManifest) {
				imageURL, err := url.Parse(machineImage.GetURL)

				params := &s3.HeadObjectInput{
					Bucket: aws.String(bucketName),
					Key:    aws.String(imageURL.Path),
				}
				headResp, err := s3Client.HeadObject(params)
				Expect(err).ToNot(HaveOccurred())

				Expect(*headResp.ServerSideEncryption).To(Equal("AES256"))

				imageURL, err = url.Parse(manifest.Parts.Part.HeadURL)

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

	testMachineImageLifecycle = func(driverConfig resources.MachineImageDriverConfig, cb ...func(resources.MachineImage)) {
		createDriver := driver.NewCreateMachineImageDriver(GinkgoWriter, creds)

		machineImage, err := createDriver.Create(driverConfig)
		Expect(err).ToNot(HaveOccurred())

		imageURL, err := url.Parse(machineImage.GetURL)

		input := &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(imageURL.Path),
		}

		_, err = s3Client.GetObject(input)
		Expect(err).ToNot(HaveOccurred())

		if len(cb) > 0 {
			cb[0](machineImage)
		}

		deleteDriver := driver.NewDeleteMachineImageDriver(GinkgoWriter, creds)

		err = deleteDriver.Delete(machineImage)
		Expect(err).ToNot(HaveOccurred())

		_, err = s3Client.GetObject(input)
		Expect(err).ToHaveOccurred())
		aerr, ok := err.(awserr.Error)
		Expect(ok).To(BeTrue())
		Expect(aerr.Code()).To(Equal(s3.ErrCodeNoSuchKey))
	}

	testMachineImageManifestLifecycle = func(driverConfig resources.MachineImageDriverConfig, cb ...func(resources.MachineImage, manifests.ImportVolumeManifest)) {
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

		Expect(m.FileFormat).To(Equal(imageFormat))
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
})
