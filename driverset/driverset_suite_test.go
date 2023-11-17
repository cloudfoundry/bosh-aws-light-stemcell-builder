package driverset_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDriverset(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Driverset Suite")
}
