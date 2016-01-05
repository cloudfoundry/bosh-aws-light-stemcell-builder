package ec2stage_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestEc2stage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ec2stage Suite")
}
