package publisher_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPublisher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Publisher Suite")
}
