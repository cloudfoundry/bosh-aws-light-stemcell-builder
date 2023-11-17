package reqinputs_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestReqinputs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reqinputs Suite")
}
