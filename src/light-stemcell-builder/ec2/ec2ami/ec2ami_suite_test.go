package ec2ami_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestEc2ami(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ec2ami Suite")
}
