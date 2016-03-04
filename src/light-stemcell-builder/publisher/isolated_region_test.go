package publisher_test

import (
	"errors"
	"light-stemcell-builder/config"
	fakeDriverset "light-stemcell-builder/driverset/fakes"
	"light-stemcell-builder/publisher"
	"light-stemcell-builder/resources"
	fakeResources "light-stemcell-builder/resources/fakes"

	. "github.com/onsi/ginkgo"
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

	It("uses the provided driver set to orchestrate the creation of an AMI", func() {
		publisherConfig := publisher.Config{
			AmiRegion: config.AmiRegion{
				RegionName: fakeRegion,
				BucketName: fakeBucketName,
			},
			AmiConfiguration: fakeAmiConfig,
		}

		fakeDs := &fakeDriverset.FakeIsolatedRegionDriverSet{}
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

		fakeMachineImageDriver := &fakeResources.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(fakeMachineImage, nil)
		fakeDs.CreateMachineImageDriverReturns(fakeMachineImageDriver)

		fakeVolumeDriver := &fakeResources.FakeVolumeDriver{}
		fakeVolumeDriver.CreateReturns(fakeVolume, nil)
		fakeDs.CreateVolumeDriverReturns(fakeVolumeDriver)

		fakeSnapshotDriver := &fakeResources.FakeSnapshotDriver{}
		fakeSnapshotDriver.CreateReturns(fakeSnapshot, nil)
		fakeDs.CreateSnapshotDriverReturns(fakeSnapshotDriver)

		fakeCreateAmiDriver := &fakeResources.FakeAmiDriver{}
		fakeCreateAmiDriver.CreateReturns(fakeAmi, nil)
		fakeDs.CreateAmiDriverReturns(fakeCreateAmiDriver)

		p := publisher.NewIsolatedRegionPublisher(GinkgoWriter, publisherConfig)
		amiCollection, err := p.Publish(fakeDs, fakeMachineImagePath)
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeDs.CreateMachineImageDriverCallCount()).To(Equal(1))
		Expect(fakeMachineImageDriver.CreateCallCount()).To(Equal(1))
		Expect(fakeMachineImageDriver.CreateArgsForCall(0)).To(Equal(resources.MachineImageDriverConfig{
			MachineImagePath: fakeMachineImagePath,
			BucketName:       fakeBucketName,
		}))

		Expect(fakeDs.CreateVolumeDriverCallCount()).To(Equal(1))
		Expect(fakeVolumeDriver.CreateCallCount()).To(Equal(1))
		Expect(fakeVolumeDriver.CreateArgsForCall(0)).To(Equal(resources.VolumeDriverConfig{
			MachineImageManifestURL: fakeMachineImageURL,
		}))

		Expect(fakeDs.CreateSnapshotDriverCallCount()).To(Equal(1))
		Expect(fakeSnapshotDriver.CreateCallCount()).To(Equal(1))
		Expect(fakeSnapshotDriver.CreateArgsForCall(0)).To(Equal(resources.SnapshotDriverConfig{
			VolumeID: fakeVolumeID,
		}))

		Expect(fakeDs.CreateAmiDriverCallCount()).To(Equal(1))
		Expect(fakeCreateAmiDriver.CreateCallCount()).To(Equal(1))
		Expect(fakeCreateAmiDriver.CreateArgsForCall(0)).To(Equal(resources.AmiDriverConfig{
			SnapshotID:    fakeSnapshotID,
			AmiProperties: fakeAmiProperties,
		}))

		Expect(amiCollection.GetAll()).To(ConsistOf(fakeAmi))
		Expect(amiCollection.VirtualizationType).To(Equal(fakeAmiConfig.VirtualizationType))
	})

	It("returns a machine image driver error if one was returned", func() {
		publisherConfig := publisher.Config{}
		fakeDs := &fakeDriverset.FakeIsolatedRegionDriverSet{}
		driverErr := errors.New("error in machine image driver")

		fakeMachineImageDriver := &fakeResources.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(resources.MachineImage{}, driverErr)
		fakeDs.CreateMachineImageDriverReturns(fakeMachineImageDriver)

		p := publisher.NewIsolatedRegionPublisher(GinkgoWriter, publisherConfig)
		_, err := p.Publish(fakeDs, fakeMachineImagePath)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(driverErr.Error()))
	})

	It("returns a volume driver error if one was returned", func() {
		publisherConfig := publisher.Config{}
		fakeDs := &fakeDriverset.FakeIsolatedRegionDriverSet{}
		driverErr := errors.New("error in volume driver")

		fakeMachineImageDriver := &fakeResources.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(resources.MachineImage{GetURL: fakeMachineImageURL}, nil)
		fakeDs.CreateMachineImageDriverReturns(fakeMachineImageDriver)

		fakeVolumeDriver := &fakeResources.FakeVolumeDriver{}
		fakeVolumeDriver.CreateReturns(resources.Volume{}, driverErr)
		fakeDs.CreateVolumeDriverReturns(fakeVolumeDriver)

		p := publisher.NewIsolatedRegionPublisher(GinkgoWriter, publisherConfig)
		_, err := p.Publish(fakeDs, fakeMachineImagePath)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(driverErr.Error()))
	})

	It("returns a snapshot driver error if one was returned", func() {
		publisherConfig := publisher.Config{}
		fakeDs := &fakeDriverset.FakeIsolatedRegionDriverSet{}
		driverErr := errors.New("error in ami driver")

		fakeMachineImageDriver := &fakeResources.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(resources.MachineImage{GetURL: fakeMachineImageURL}, nil)
		fakeDs.CreateMachineImageDriverReturns(fakeMachineImageDriver)

		fakeVolumeDriver := &fakeResources.FakeVolumeDriver{}
		fakeVolumeDriver.CreateReturns(resources.Volume{ID: fakeVolumeID}, nil)
		fakeDs.CreateVolumeDriverReturns(fakeVolumeDriver)

		fakeSnapshotDriver := &fakeResources.FakeSnapshotDriver{}
		fakeSnapshotDriver.CreateReturns(resources.Snapshot{}, driverErr)
		fakeDs.CreateSnapshotDriverReturns(fakeSnapshotDriver)

		p := publisher.NewIsolatedRegionPublisher(GinkgoWriter, publisherConfig)
		_, err := p.Publish(fakeDs, fakeMachineImagePath)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(driverErr.Error()))
	})

	It("returns a create ami driver error if one was returned", func() {
		publisherConfig := publisher.Config{}
		fakeDs := &fakeDriverset.FakeIsolatedRegionDriverSet{}
		driverErr := errors.New("error in create ami driver")

		fakeMachineImageDriver := &fakeResources.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(resources.MachineImage{GetURL: fakeMachineImageURL}, nil)
		fakeDs.CreateMachineImageDriverReturns(fakeMachineImageDriver)

		fakeVolumeDriver := &fakeResources.FakeVolumeDriver{}
		fakeVolumeDriver.CreateReturns(resources.Volume{ID: fakeVolumeID}, nil)
		fakeDs.CreateVolumeDriverReturns(fakeVolumeDriver)

		fakeSnapshotDriver := &fakeResources.FakeSnapshotDriver{}
		fakeSnapshotDriver.CreateReturns(resources.Snapshot{ID: fakeSnapshotID}, nil)
		fakeDs.CreateSnapshotDriverReturns(fakeSnapshotDriver)

		fakeAmiDriver := &fakeResources.FakeAmiDriver{}
		fakeAmiDriver.CreateReturns(resources.Ami{}, driverErr)
		fakeDs.CreateAmiDriverReturns(fakeAmiDriver)

		p := publisher.NewIsolatedRegionPublisher(GinkgoWriter, publisherConfig)
		_, err := p.Publish(fakeDs, fakeMachineImagePath)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(driverErr.Error()))
	})
})
