package driverset_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDriversets(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Driversets Suite")
}
