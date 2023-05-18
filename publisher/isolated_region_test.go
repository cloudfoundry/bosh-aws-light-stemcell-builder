package publisher_test

import (
	"errors"

	"light-stemcell-builder/config"
	"light-stemcell-builder/driverset/driversetfakes"
	"light-stemcell-builder/publisher"
	"light-stemcell-builder/resources"
	"light-stemcell-builder/resources/resourcesfakes"

	"github.com/aws/aws-sdk-go/aws/session"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IsolatedRegionPublisher", func() {
	const (
		fakeMachineImageURL  = "fake machine image url"
		fakeVolumeID         = "fake volume id"
		fakeSnapshotID       = "fake snapshot id"
		fakeAmiID            = "fake AMI id"
		fakeBucketName       = "fake bucket name"
		fakeRegion           = "fake region"
		fakeMachineImagePath = "fake machine image path"
		fakeVolumeSizeGB     = 3
	)

	var fakeAmiConfig = config.AmiConfiguration{
		Visibility:         "public",
		Description:        "fake ami description",
		AmiName:            "fake ami name",
		VirtualizationType: "fake virtualization type",
	}
	var fakeAmiProperties = resources.AmiProperties{
		Name:               fakeAmiConfig.AmiName,
		Description:        fakeAmiConfig.Description,
		Accessibility:      fakeAmiConfig.Visibility,
		VirtualizationType: fakeAmiConfig.VirtualizationType,
	}

	var awsSession = session.Must(session.NewSession())

	It("uses the provided driver set to orchestrate the creation of an AMI", func() {
		publisherConfig := publisher.Config{
			AmiRegion: config.AmiRegion{
				RegionName: fakeRegion,
				BucketName: fakeBucketName,
			},
			AmiConfiguration: fakeAmiConfig,
		}
		machineImageConfig := publisher.MachineImageConfig{
			LocalPath:    fakeMachineImagePath,
			FileFormat:   resources.VolumeRawFormat,
			VolumeSizeGB: fakeVolumeSizeGB,
		}

		fakeDs := &driversetfakes.FakeIsolatedRegionDriverSet{}
		fakeMachineImage := resources.MachineImage{
			GetURL: fakeMachineImageURL,
		}

		fakeVolume := resources.Volume{
			ID: fakeVolumeID,
		}

		fakeSnapshot := resources.Snapshot{
			ID: fakeSnapshotID,
		}

		fakeAmi := resources.Ami{
			ID:     fakeAmiID,
			Region: fakeRegion,
		}

		fakeMachineImageDriver := &resourcesfakes.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(fakeMachineImage, nil)
		fakeMachineImageDriver.DeleteReturns(nil)
		fakeDs.MachineImageDriverReturns(fakeMachineImageDriver)

		fakeVolumeDriver := &resourcesfakes.FakeVolumeDriver{}
		fakeVolumeDriver.CreateReturns(fakeVolume, nil)
		fakeVolumeDriver.DeleteReturns(nil)
		fakeDs.VolumeDriverReturns(fakeVolumeDriver)

		fakeSnapshotDriver := &resourcesfakes.FakeSnapshotDriver{}
		fakeSnapshotDriver.CreateReturns(fakeSnapshot, nil)
		fakeDs.CreateSnapshotDriverReturns(fakeSnapshotDriver)

		fakeCreateAmiDriver := &resourcesfakes.FakeAmiDriver{}
		fakeCreateAmiDriver.CreateReturns(fakeAmi, nil)
		fakeDs.CreateAmiDriverReturns(fakeCreateAmiDriver)

		p := publisher.NewIsolatedRegionPublisher(GinkgoWriter, awsSession, publisherConfig)
		amiCollection, err := p.Publish(fakeDs, machineImageConfig)
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeDs.MachineImageDriverCallCount()).To(Equal(1), "Expected Driverset.MachineImageDriver to be called once")
		Expect(fakeMachineImageDriver.CreateCallCount()).To(Equal(1), "Expected MachineImageDriver.Create to be called once")
		Expect(fakeMachineImageDriver.CreateArgsForCall(0)).To(Equal(resources.MachineImageDriverConfig{
			MachineImagePath: fakeMachineImagePath,
			BucketName:       fakeBucketName,
			FileFormat:       machineImageConfig.FileFormat,
			VolumeSizeGB:     fakeVolumeSizeGB,
		}))

		Expect(fakeDs.VolumeDriverCallCount()).To(Equal(1), "Expected Driverset.VolumeDriver to be called once")
		Expect(fakeVolumeDriver.CreateCallCount()).To(Equal(1), "Expected VolumeDriver.Create to be called once")
		Expect(fakeVolumeDriver.CreateArgsForCall(0)).To(Equal(resources.VolumeDriverConfig{
			MachineImageManifestURL: fakeMachineImageURL,
		}))

		Expect(fakeDs.CreateSnapshotDriverCallCount()).To(Equal(1), "Expected Driverset.CreateSnapshotDriver to be called once")
		Expect(fakeSnapshotDriver.CreateCallCount()).To(Equal(1), "Expected CreateSnapshotDriver.Create to be called once")
		Expect(fakeSnapshotDriver.CreateArgsForCall(0)).To(Equal(resources.SnapshotDriverConfig{
			VolumeID: fakeVolumeID,
		}))

		Expect(fakeDs.CreateAmiDriverCallCount()).To(Equal(1), "Expected Driverset.CreateAmiDriver to be called once")
		Expect(fakeCreateAmiDriver.CreateCallCount()).To(Equal(1), "Expected CreateAmiDriver.Create to be called once")
		Expect(fakeCreateAmiDriver.CreateArgsForCall(0)).To(Equal(resources.AmiDriverConfig{
			SnapshotID:    fakeSnapshotID,
			AmiProperties: fakeAmiProperties,
		}))

		Expect(fakeMachineImageDriver.DeleteCallCount()).To(Equal(1), "Expected MachineImageDriver.Delete to be called once")
		Expect(fakeMachineImageDriver.DeleteArgsForCall(0)).To(Equal(fakeMachineImage))

		Expect(fakeVolumeDriver.DeleteCallCount()).To(Equal(1), "Expected VolumeDriver.Delete to be called once")
		Expect(fakeVolumeDriver.DeleteArgsForCall(0)).To(Equal(fakeVolume))

		Expect(amiCollection.GetAll()).To(ConsistOf(fakeAmi))
		Expect(amiCollection.VirtualizationType).To(Equal(fakeAmiConfig.VirtualizationType))
	})

	It("returns a machine image driver error if one was returned", func() {
		publisherConfig := publisher.Config{}
		machineImageConfig := publisher.MachineImageConfig{}
		fakeDs := &driversetfakes.FakeIsolatedRegionDriverSet{}
		driverErr := errors.New("error in machine image driver")

		fakeMachineImageDriver := &resourcesfakes.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(resources.MachineImage{}, driverErr)
		fakeDs.MachineImageDriverReturns(fakeMachineImageDriver)

		p := publisher.NewIsolatedRegionPublisher(GinkgoWriter, awsSession, publisherConfig)
		_, err := p.Publish(fakeDs, machineImageConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(driverErr.Error()))
	})

	It("returns a volume driver error if one was returned", func() {
		publisherConfig := publisher.Config{}
		machineImageConfig := publisher.MachineImageConfig{}
		fakeDs := &driversetfakes.FakeIsolatedRegionDriverSet{}
		driverErr := errors.New("error in volume driver")

		fakeMachineImageDriver := &resourcesfakes.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(resources.MachineImage{GetURL: fakeMachineImageURL}, nil)
		fakeDs.MachineImageDriverReturns(fakeMachineImageDriver)

		fakeVolumeDriver := &resourcesfakes.FakeVolumeDriver{}
		fakeVolumeDriver.CreateReturns(resources.Volume{}, driverErr)
		fakeDs.VolumeDriverReturns(fakeVolumeDriver)

		p := publisher.NewIsolatedRegionPublisher(GinkgoWriter, awsSession, publisherConfig)
		_, err := p.Publish(fakeDs, machineImageConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(driverErr.Error()))
	})

	It("returns a snapshot driver error if one was returned", func() {
		publisherConfig := publisher.Config{}
		machineImageConfig := publisher.MachineImageConfig{}
		fakeDs := &driversetfakes.FakeIsolatedRegionDriverSet{}
		driverErr := errors.New("error in ami driver")

		fakeMachineImageDriver := &resourcesfakes.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(resources.MachineImage{GetURL: fakeMachineImageURL}, nil)
		fakeDs.MachineImageDriverReturns(fakeMachineImageDriver)

		fakeVolumeDriver := &resourcesfakes.FakeVolumeDriver{}
		fakeVolumeDriver.CreateReturns(resources.Volume{ID: fakeVolumeID}, nil)
		fakeDs.VolumeDriverReturns(fakeVolumeDriver)

		fakeSnapshotDriver := &resourcesfakes.FakeSnapshotDriver{}
		fakeSnapshotDriver.CreateReturns(resources.Snapshot{}, driverErr)
		fakeDs.CreateSnapshotDriverReturns(fakeSnapshotDriver)

		p := publisher.NewIsolatedRegionPublisher(GinkgoWriter, awsSession, publisherConfig)
		_, err := p.Publish(fakeDs, machineImageConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(driverErr.Error()))
	})

	It("returns a create ami driver error if one was returned", func() {
		publisherConfig := publisher.Config{}
		machineImageConfig := publisher.MachineImageConfig{}
		fakeDs := &driversetfakes.FakeIsolatedRegionDriverSet{}
		driverErr := errors.New("error in create ami driver")

		fakeMachineImageDriver := &resourcesfakes.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(resources.MachineImage{GetURL: fakeMachineImageURL}, nil)
		fakeDs.MachineImageDriverReturns(fakeMachineImageDriver)

		fakeVolumeDriver := &resourcesfakes.FakeVolumeDriver{}
		fakeVolumeDriver.CreateReturns(resources.Volume{ID: fakeVolumeID}, nil)
		fakeDs.VolumeDriverReturns(fakeVolumeDriver)

		fakeSnapshotDriver := &resourcesfakes.FakeSnapshotDriver{}
		fakeSnapshotDriver.CreateReturns(resources.Snapshot{ID: fakeSnapshotID}, nil)
		fakeDs.CreateSnapshotDriverReturns(fakeSnapshotDriver)

		fakeAmiDriver := &resourcesfakes.FakeAmiDriver{}
		fakeAmiDriver.CreateReturns(resources.Ami{}, driverErr)
		fakeDs.CreateAmiDriverReturns(fakeAmiDriver)

		p := publisher.NewIsolatedRegionPublisher(GinkgoWriter, awsSession, publisherConfig)
		_, err := p.Publish(fakeDs, machineImageConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(driverErr.Error()))
	})
})
