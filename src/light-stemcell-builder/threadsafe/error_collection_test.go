package threadsafe_test

import (
	"errors"
	. "light-stemcell-builder/threadsafe"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ErrorCollection", func() {
	It("outputs an error with expected messages for all the errors collected", func() {
		e := NewErrorCollection()
		e.AddError("Label 1", errors.New("The quick brown"))
		e.AddError("Label 2", errors.New("fox jumps over"))
		e.AddError("Label 3", errors.New("the lazy dog"))
		err := e.BuildError()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(MatchRegexp("(?m)^Label 1: The quick brown$"))
		Expect(err.Error()).To(MatchRegexp("(?m)^Label 2: fox jumps over$"))
		Expect(err.Error()).To(MatchRegexp("(?m)^Label 3: the lazy dog$"))
	})
	It("does not output an error when no errors have been added", func() {
		e := NewErrorCollection()
		err := e.BuildError()
		Expect(err).ToNot(HaveOccurred())
	})
})
