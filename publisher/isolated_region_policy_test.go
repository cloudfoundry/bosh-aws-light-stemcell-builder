package publisher_test

import (
	"errors"
	"io"
	"light-stemcell-builder/config"
	"light-stemcell-builder/driverset/driversetfakes"
	"light-stemcell-builder/publisher"
	"light-stemcell-builder/resources"
	"light-stemcell-builder/resources/resourcesfakes"
	"strings"
	"testing"
)

func TestIsolatedEncryptionRejectedBeforeResourceCreation(t *testing.T) {
	for _, amiConfig := range []config.AmiConfiguration{{Encrypted: true}, {KmsKeyId: "requested-key"}} {
		p := publisher.NewIsolatedRegionPublisher(io.Discard, publisher.Config{AmiConfiguration: amiConfig})
		ds := &driversetfakes.FakeIsolatedRegionDriverSet{}
		imageDriver := &resourcesfakes.FakeMachineImageDriver{}
		imageDriver.CreateReturns(resources.MachineImage{}, errors.New("must reject before creating image"))
		ds.MachineImageDriverReturns(imageDriver)
		_, err := p.Publish(ds, publisher.MachineImageConfig{})
		if err == nil || !strings.Contains(err.Error(), "not supported") {
			t.Fatalf("expected explicit unsupported encryption error, got %v", err)
		}
		if ds.MachineImageDriverCallCount() != 0 {
			t.Fatal("created resources before rejecting unsupported encryption")
		}
	}
}
