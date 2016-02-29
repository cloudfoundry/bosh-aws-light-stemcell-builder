package reqinputs_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestReqinputs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reqinputs Suite")
}
