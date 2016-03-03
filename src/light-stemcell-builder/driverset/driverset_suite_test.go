package driverset_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDriverset(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Driverset Suite")
}
