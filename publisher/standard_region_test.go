package publisher_test

import (
	"errors"

	"light-stemcell-builder/config"
	"light-stemcell-builder/driverset/driversetfakes"
	"light-stemcell-builder/publisher"
	"light-stemcell-builder/resources"
	"light-stemcell-builder/resources/resourcesfakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("StandardRegionPublisher", func() {

	const (
		fakeMachineImageURL  = "fake machine image url"
		fakeSnapshotID       = "fake snapshot id"
		fakeAmiID            = "fake AMI id"
		fakeCopiedAmiID      = "fake copied AMI id"
		fakeBucketName       = "fake bucket name"
		fakeRegion           = "fake region"
		fakeMachineImagePath = "fake machine image path"
		fakeCopyDestination  = "fake copy destination"
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
				RegionName:   fakeRegion,
				BucketName:   fakeBucketName,
				Destinations: []string{fakeCopyDestination},
			},
			AmiConfiguration: fakeAmiConfig,
		}
		machineImageConfig := publisher.MachineImageConfig{
			LocalPath:  fakeMachineImagePath,
			FileFormat: resources.VolumeRawFormat,
		}

		fakeDs := &driversetfakes.FakeStandardRegionDriverSet{}
		fakeMachineImage := resources.MachineImage{
			GetURL: fakeMachineImageURL,
		}
		fakeSnapshot := resources.Snapshot{
			ID: fakeSnapshotID,
		}
		fakeAmi := resources.Ami{
			ID:     fakeAmiID,
			Region: fakeRegion,
		}
		fakeCopiedAmi := resources.Ami{
			ID:     fakeCopiedAmiID,
			Region: fakeCopyDestination,
		}

		fakeMachineImageDriver := &resourcesfakes.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(fakeMachineImage, nil)
		fakeMachineImageDriver.DeleteReturns(nil)
		fakeDs.MachineImageDriverReturns(fakeMachineImageDriver)

		fakeSnapshotDriver := &resourcesfakes.FakeSnapshotDriver{}
		fakeSnapshotDriver.CreateReturns(fakeSnapshot, nil)
		fakeDs.CreateSnapshotDriverReturns(fakeSnapshotDriver)

		fakeCreateAmiDriver := &resourcesfakes.FakeAmiDriver{}
		fakeCreateAmiDriver.CreateReturns(fakeAmi, nil)
		fakeDs.CreateAmiDriverReturns(fakeCreateAmiDriver)

		fakeCopyAmiDriver := &resourcesfakes.FakeAmiDriver{}
		fakeCopyAmiDriver.CreateReturns(fakeCopiedAmi, nil)
		fakeDs.CopyAmiDriverReturns(fakeCopyAmiDriver)

		p := publisher.NewStandardRegionPublisher(GinkgoWriter, publisherConfig)
		amiCollection, err := p.Publish(fakeDs, machineImageConfig)
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeDs.MachineImageDriverCallCount()).To(Equal(1), "Expected Driverset.MachineImageDriver to be called once")
		Expect(fakeMachineImageDriver.CreateCallCount()).To(Equal(1), "Expected MachineImageDriver.Create to be called once")
		Expect(fakeMachineImageDriver.CreateArgsForCall(0)).To(Equal(resources.MachineImageDriverConfig{
			MachineImagePath: fakeMachineImagePath,
			FileFormat:       resources.VolumeRawFormat,
			BucketName:       fakeBucketName,
		}))

		Expect(fakeDs.CreateSnapshotDriverCallCount()).To(Equal(1), "Expected Driverset.CreateSnapshotDriver to be called once")
		Expect(fakeSnapshotDriver.CreateCallCount()).To(Equal(1), "Expected CreateSnapshotDriver.Create to be called once")
		Expect(fakeSnapshotDriver.CreateArgsForCall(0)).To(Equal(resources.SnapshotDriverConfig{
			MachineImageURL: fakeMachineImageURL,
			FileFormat:      resources.VolumeRawFormat,
		}))

		Expect(fakeDs.CreateAmiDriverCallCount()).To(Equal(1), "Expected Driverset.CreateAmiDriver to be called once")
		Expect(fakeCreateAmiDriver.CreateCallCount()).To(Equal(1), "Expected CreateAmiDriver.Create to be called once")
		Expect(fakeCreateAmiDriver.CreateArgsForCall(0)).To(Equal(resources.AmiDriverConfig{
			SnapshotID:    fakeSnapshotID,
			AmiProperties: fakeAmiProperties,
		}))

		Expect(fakeDs.CopyAmiDriverCallCount()).To(Equal(1), "Expected Driverset.CopyAmiDriver to be called once")
		Expect(fakeCopyAmiDriver.CreateCallCount()).To(Equal(1), "Expected CopyAmiDriver.Create to be called once")

		Expect(fakeCopyAmiDriver.CreateArgsForCall(0)).To(Equal(resources.AmiDriverConfig{
			ExistingAmiID:     fakeAmiID,
			DestinationRegion: fakeCopyDestination,
			AmiProperties:     fakeAmiProperties,
		}))

		Expect(fakeMachineImageDriver.DeleteCallCount()).To(Equal(1), "Expected MachineImageDriver.Delete to be called once")
		Expect(fakeMachineImageDriver.DeleteArgsForCall(0)).To(Equal(fakeMachineImage))

		Expect(amiCollection.GetAll()).To(ConsistOf(fakeAmi, fakeCopiedAmi))
		Expect(amiCollection.VirtualizationType).To(Equal(fakeAmiConfig.VirtualizationType))
	})

	It("returns a machine image driver error if one was returned", func() {
		publisherConfig := publisher.Config{}
		machineImageConfig := publisher.MachineImageConfig{}

		fakeDs := &driversetfakes.FakeStandardRegionDriverSet{}

		driverErr := errors.New("error in machine image driver")

		fakeMachineImageDriver := &resourcesfakes.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(resources.MachineImage{}, driverErr)
		fakeDs.MachineImageDriverReturns(fakeMachineImageDriver)

		p := publisher.NewStandardRegionPublisher(GinkgoWriter, publisherConfig)
		_, err := p.Publish(fakeDs, machineImageConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(driverErr.Error()))
	})

	It("returns a snapshot driver error if one was returned", func() {
		publisherConfig := publisher.Config{}
		machineImageConfig := publisher.MachineImageConfig{}

		fakeDs := &driversetfakes.FakeStandardRegionDriverSet{}
		fakeMachineImage := resources.MachineImage{
			GetURL: fakeMachineImageURL,
		}

		driverErr := errors.New("error in ami driver")

		fakeMachineImageDriver := &resourcesfakes.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(fakeMachineImage, nil)
		fakeDs.MachineImageDriverReturns(fakeMachineImageDriver)

		fakeSnapshotDriver := &resourcesfakes.FakeSnapshotDriver{}
		fakeSnapshotDriver.CreateReturns(resources.Snapshot{}, driverErr)
		fakeDs.CreateSnapshotDriverReturns(fakeSnapshotDriver)

		p := publisher.NewStandardRegionPublisher(GinkgoWriter, publisherConfig)
		_, err := p.Publish(fakeDs, machineImageConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(driverErr.Error()))
	})

	It("returns a create ami driver error if one was returned", func() {
		publisherConfig := publisher.Config{}
		machineImageConfig := publisher.MachineImageConfig{}

		fakeDs := &driversetfakes.FakeStandardRegionDriverSet{}
		fakeMachineImage := resources.MachineImage{
			GetURL: fakeMachineImageURL,
		}
		fakeSnapshot := resources.Snapshot{
			ID: fakeSnapshotID,
		}

		driverErr := errors.New("error in create ami driver")

		fakeMachineImageDriver := &resourcesfakes.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(fakeMachineImage, nil)
		fakeDs.MachineImageDriverReturns(fakeMachineImageDriver)

		fakeSnapshotDriver := &resourcesfakes.FakeSnapshotDriver{}
		fakeSnapshotDriver.CreateReturns(fakeSnapshot, nil)
		fakeDs.CreateSnapshotDriverReturns(fakeSnapshotDriver)

		fakeAmiDriver := &resourcesfakes.FakeAmiDriver{}
		fakeAmiDriver.CreateReturns(resources.Ami{}, driverErr)
		fakeDs.CreateAmiDriverReturns(fakeAmiDriver)

		p := publisher.NewStandardRegionPublisher(GinkgoWriter, publisherConfig)
		_, err := p.Publish(fakeDs, machineImageConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(driverErr.Error()))
	})

	It("returns a copy ami driver error if one was returned", func() {
		publisherConfig := publisher.Config{
			AmiRegion: config.AmiRegion{
				Destinations: []string{fakeCopyDestination},
			},
			AmiConfiguration: fakeAmiConfig,
		}
		machineImageConfig := publisher.MachineImageConfig{}

		fakeDs := &driversetfakes.FakeStandardRegionDriverSet{}
		fakeMachineImage := resources.MachineImage{
			GetURL: fakeMachineImageURL,
		}
		fakeSnapshot := resources.Snapshot{
			ID: fakeSnapshotID,
		}

		driverErr := errors.New("error in copy ami driver")

		fakeMachineImageDriver := &resourcesfakes.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(fakeMachineImage, nil)
		fakeDs.MachineImageDriverReturns(fakeMachineImageDriver)

		fakeSnapshotDriver := &resourcesfakes.FakeSnapshotDriver{}
		fakeSnapshotDriver.CreateReturns(fakeSnapshot, nil)
		fakeDs.CreateSnapshotDriverReturns(fakeSnapshotDriver)

		fakeAmi := resources.Ami{
			ID:     fakeAmiID,
			Region: fakeRegion,
		}

		fakeCreateAmiDriver := &resourcesfakes.FakeAmiDriver{}
		fakeCreateAmiDriver.CreateReturns(fakeAmi, nil)
		fakeDs.CreateAmiDriverReturns(fakeCreateAmiDriver)

		fakeCopyAmiDriver := &resourcesfakes.FakeAmiDriver{}
		fakeCopyAmiDriver.CreateReturns(resources.Ami{}, driverErr)
		fakeDs.CopyAmiDriverReturns(fakeCopyAmiDriver)

		p := publisher.NewStandardRegionPublisher(GinkgoWriter, publisherConfig)
		_, err := p.Publish(fakeDs, machineImageConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(driverErr.Error()))
	})
})
