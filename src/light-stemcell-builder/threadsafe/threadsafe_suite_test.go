package threadsafe_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestThreadsafe(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Threadsafe Suite")
}
