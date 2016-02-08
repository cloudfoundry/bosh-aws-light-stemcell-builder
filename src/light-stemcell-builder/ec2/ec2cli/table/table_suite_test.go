package table_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestEc2table(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ec2 Table Suite")
}
