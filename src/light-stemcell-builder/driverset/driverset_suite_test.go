package driverset_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDriverset(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Driverset Suite")
}
