package drivers_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDrivers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Drivers Suite")
}
