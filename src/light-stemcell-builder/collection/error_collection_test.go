package collection_test

import (
	"errors"

	"light-stemcell-builder/collection"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Error", func() {
	It("outputs an error with expected messages for all the errors collected", func() {
		e := collection.Error{}
		e.Add(errors.New("The quick brown"))
		e.Add(errors.New("fox jumps over"))
		e.Add(errors.New("the lazy dog"))
		err := e.Error()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("The quick brown"))
		Expect(err.Error()).To(ContainSubstring("fox jumps over"))
		Expect(err.Error()).To(ContainSubstring("the lazy dog"))
	})

	It("does not output an error when no errors have been added", func() {
		e := collection.Error{}
		err := e.Error()
		Expect(err).ToNot(HaveOccurred())
	})
})
