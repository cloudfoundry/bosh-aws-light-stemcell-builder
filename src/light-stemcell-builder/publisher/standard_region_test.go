package publisher_test

import (
	"errors"
	fakeDriversets "light-stemcell-builder/driverset/fakes"
	"light-stemcell-builder/publisher"
	"light-stemcell-builder/resources"
	fakeResources "light-stemcell-builder/resources/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StandardRegionPublisher", func() {
	It("can be initialized with publisher configuration", func() {
		conf := publisher.Config{}

		p := publisher.NewStandardRegionPublisher(conf)
		Expect(p).ToNot(BeNil())
	})

	It("uses the provided driver set to orchestrate the creation of an AMI", func() {
		conf := publisher.Config{
			MachineImagePath: fakeMachineImagePath,
			BucketName:       fakeBucketName,
			AmiProperties:    fakeAmiProperties,
			CopyDestinations: []string{fakeCopyDestination},
		}

		fakeDs := &fakedriverset.FakeStandardRegionDriverSet{}
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

		fakeMachineImageDriver := &fakeResources.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(fakeMachineImage, nil)
		fakeDs.CreateMachineImageDriverReturns(fakeMachineImageDriver)

		fakeSnapshotDriver := &fakeResources.FakeSnapshotDriver{}
		fakeSnapshotDriver.CreateReturns(fakeSnapshot, nil)
		fakeDs.CreateSnapshotDriverReturns(fakeSnapshotDriver)

		fakeCreateAmiDriver := &fakeResources.FakeAmiDriver{}
		fakeCreateAmiDriver.CreateReturns(fakeAmi, nil)
		fakeDs.CreateAmiDriverReturns(fakeCreateAmiDriver)

		fakeCopyAmiDriver := &fakeResources.FakeAmiDriver{}
		fakeCopyAmiDriver.CreateReturns(fakeCopiedAmi, nil)
		fakeDs.CopyAmiDriverReturns(fakeCopyAmiDriver)

		p := publisher.NewStandardRegionPublisher(conf)
		amiCollection, err := p.Publish(fakeDs)
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeDs.CreateMachineImageDriverCallCount()).To(Equal(1))
		Expect(fakeMachineImageDriver.CreateCallCount()).To(Equal(1))
		Expect(fakeMachineImageDriver.CreateArgsForCall(0)).To(Equal(resources.MachineImageDriverConfig{
			MachineImagePath: fakeMachineImagePath,
			BucketName:       fakeBucketName,
		}))

		Expect(fakeDs.CreateSnapshotDriverCallCount()).To(Equal(1))
		Expect(fakeSnapshotDriver.CreateCallCount()).To(Equal(1))
		Expect(fakeSnapshotDriver.CreateArgsForCall(0)).To(Equal(resources.SnapshotDriverConfig{
			MachineImageURL: fakeMachineImageURL,
		}))

		Expect(fakeDs.CreateAmiDriverCallCount()).To(Equal(1))
		Expect(fakeCreateAmiDriver.CreateCallCount()).To(Equal(1))
		Expect(fakeCreateAmiDriver.CreateArgsForCall(0)).To(Equal(resources.AmiDriverConfig{
			SnapshotID:    fakeSnapshotID,
			AmiProperties: fakeAmiProperties,
		}))

		Expect(fakeDs.CopyAmiDriverCallCount()).To(Equal(1))
		Expect(fakeCopyAmiDriver.CreateCallCount()).To(Equal(1))

		Expect(fakeCopyAmiDriver.CreateArgsForCall(0)).To(Equal(resources.AmiDriverConfig{
			ExistingAmiID:     fakeAmiID,
			DestinationRegion: fakeCopyDestination,
			AmiProperties:     fakeAmiProperties,
		}))

		Expect(amiCollection.GetAll()).To(ConsistOf(fakeAmi, fakeCopiedAmi))
	})

	It("returns a machine image driver error if one was returned", func() {
		conf := publisher.Config{}

		fakeDs := &fakedriverset.FakeStandardRegionDriverSet{}

		driverErr := errors.New("error in machine image driver")

		fakeMachineImageDriver := &fakeResources.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(resources.MachineImage{}, driverErr)
		fakeDs.CreateMachineImageDriverReturns(fakeMachineImageDriver)

		p := publisher.NewStandardRegionPublisher(conf)
		_, err := p.Publish(fakeDs)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(driverErr.Error()))
	})

	It("returns a snapshot driver error if one was returned", func() {
		conf := publisher.Config{}

		fakeDs := &fakedriverset.FakeStandardRegionDriverSet{}
		fakeMachineImage := resources.MachineImage{
			GetURL: fakeMachineImageURL,
		}

		driverErr := errors.New("error in ami driver")

		fakeMachineImageDriver := &fakeResources.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(fakeMachineImage, nil)
		fakeDs.CreateMachineImageDriverReturns(fakeMachineImageDriver)

		fakeSnapshotDriver := &fakeResources.FakeSnapshotDriver{}
		fakeSnapshotDriver.CreateReturns(resources.Snapshot{}, driverErr)
		fakeDs.CreateSnapshotDriverReturns(fakeSnapshotDriver)

		p := publisher.NewStandardRegionPublisher(conf)
		_, err := p.Publish(fakeDs)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(driverErr.Error()))
	})

	It("returns a create ami driver error if one was returned", func() {
		conf := publisher.Config{}

		fakeDs := &fakedriverset.FakeStandardRegionDriverSet{}
		fakeMachineImage := resources.MachineImage{
			GetURL: fakeMachineImageURL,
		}
		fakeSnapshot := resources.Snapshot{
			ID: fakeSnapshotID,
		}

		driverErr := errors.New("error in create ami driver")

		fakeMachineImageDriver := &fakeResources.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(fakeMachineImage, nil)
		fakeDs.CreateMachineImageDriverReturns(fakeMachineImageDriver)

		fakeSnapshotDriver := &fakeResources.FakeSnapshotDriver{}
		fakeSnapshotDriver.CreateReturns(fakeSnapshot, nil)
		fakeDs.CreateSnapshotDriverReturns(fakeSnapshotDriver)

		fakeAmiDriver := &fakeResources.FakeAmiDriver{}
		fakeAmiDriver.CreateReturns(resources.Ami{}, driverErr)
		fakeDs.CreateAmiDriverReturns(fakeAmiDriver)

		p := publisher.NewStandardRegionPublisher(conf)
		_, err := p.Publish(fakeDs)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(driverErr.Error()))
	})

	It("returns a copy ami driver error if one was returned", func() {
		conf := publisher.Config{
			CopyDestinations: []string{fakeCopyDestination},
		}

		fakeDs := &fakedriverset.FakeStandardRegionDriverSet{}
		fakeMachineImage := resources.MachineImage{
			GetURL: fakeMachineImageURL,
		}
		fakeSnapshot := resources.Snapshot{
			ID: fakeSnapshotID,
		}

		driverErr := errors.New("error in copy ami driver")

		fakeMachineImageDriver := &fakeResources.FakeMachineImageDriver{}
		fakeMachineImageDriver.CreateReturns(fakeMachineImage, nil)
		fakeDs.CreateMachineImageDriverReturns(fakeMachineImageDriver)

		fakeSnapshotDriver := &fakeResources.FakeSnapshotDriver{}
		fakeSnapshotDriver.CreateReturns(fakeSnapshot, nil)
		fakeDs.CreateSnapshotDriverReturns(fakeSnapshotDriver)

		fakeAmi := resources.Ami{
			ID:     fakeAmiID,
			Region: fakeRegion,
		}

		fakeCreateAmiDriver := &fakeResources.FakeAmiDriver{}
		fakeCreateAmiDriver.CreateReturns(fakeAmi, nil)
		fakeDs.CreateAmiDriverReturns(fakeCreateAmiDriver)

		fakeCopyAmiDriver := &fakeResources.FakeAmiDriver{}
		fakeCopyAmiDriver.CreateReturns(resources.Ami{}, driverErr)
		fakeDs.CopyAmiDriverReturns(fakeCopyAmiDriver)

		p := publisher.NewStandardRegionPublisher(conf)
		_, err := p.Publish(fakeDs)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(driverErr.Error()))
	})
})

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

var fakeAmiProperties = resources.AmiProperties{
	Accessibility:      "fake ami accessability",
	Description:        "fake ami description",
	Name:               "fake ami name",
	VirtualizationType: "fake virtualization type",
}
