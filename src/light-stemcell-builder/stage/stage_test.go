package stage_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"light-stemcell-builder/stage"
	"log"
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type mockRunner func(data int) (int, error)
type mockUndoer func(data int) (int, error)

type mockStage struct {
	run        mockRunner
	undo       mockUndoer
	outputData int
}

func (s *mockStage) Run(logger *log.Logger, data interface{}) (interface{}, error) {
	var dummyInt int
	if reflect.TypeOf(data) != reflect.TypeOf(dummyInt) {
		return nil, fmt.Errorf("stage: mockStage expected type int, got: %s", reflect.TypeOf(data))
	}
	var err error
	s.outputData, err = s.run(data.(int))
	return s.outputData, err
}

func (s *mockStage) Rollback(logger *log.Logger) error {
	var err error
	s.outputData, err = s.undo(s.outputData)
	return err
}

func (s *mockStage) Name() string {
	return "MockStage"
}

var _ = Describe("Stage", func() {
	Describe("RunStages", func() {
		noopUndoer := func(data int) (int, error) {
			return data, nil
		}

		nullLogger := log.New(ioutil.Discard, "", log.LstdFlags)

		It("takes in initial data as input", func() {
			var runInput int
			dummyRunner := func(data int) (int, error) {
				runInput = data
				return data, nil
			}
			mockStage := &mockStage{run: dummyRunner, undo: noopUndoer}
			stages := []stage.Stage{mockStage}
			_, err := stage.RunStages(nullLogger, stages, 42)
			Expect(err).ToNot(HaveOccurred())
			Expect(runInput).To(Equal(42))
		})

		It("outputs data from the runner", func() {
			var dummyInt int
			dummyRunner := func(data int) (int, error) {
				return 42, nil
			}
			mockStage := &mockStage{run: dummyRunner, undo: noopUndoer}
			stages := []stage.Stage{mockStage}
			stagesOutput, err := stage.RunStages(nullLogger, stages, 500)
			lastOutput := stagesOutput[len(stagesOutput)-1]
			Expect(err).ToNot(HaveOccurred())
			Expect(lastOutput).To(BeAssignableToTypeOf(dummyInt))
			Expect(lastOutput.(int)).To(Equal(42))
		})

		It("chains together multiple stages, passing data from one to the next", func() {
			var dummyInt int
			dummyRunner := func(data int) (int, error) {
				return data + 1, nil
			}
			mockStage1 := &mockStage{run: dummyRunner, undo: noopUndoer}
			mockStage2 := &mockStage{run: dummyRunner, undo: noopUndoer}
			mockStage3 := &mockStage{run: dummyRunner, undo: noopUndoer}
			stages := []stage.Stage{mockStage1, mockStage2, mockStage3}
			stagesOutput, err := stage.RunStages(nullLogger, stages, 42)
			Expect(err).ToNot(HaveOccurred())
			for i := range stagesOutput {
				output := stagesOutput[i]
				Expect(output).To(BeAssignableToTypeOf(dummyInt))
				Expect(output.(int)).To(Equal(43 + i))
			}
		})

		It("rollsback previous stages when a stage has failed, and outputs the error", func() {
			dummyRunner := func(data int) (int, error) {
				return data + 1, nil
			}
			errorRunner := func(data int) (int, error) {
				return 0, fmt.Errorf("this is an error")
			}
			dummyUndoer := func(data int) (int, error) {
				return data - 2, nil
			}
			mockStage1 := &mockStage{run: dummyRunner, undo: dummyUndoer}
			mockStage2 := &mockStage{run: dummyRunner, undo: dummyUndoer}
			errorStage := &mockStage{run: errorRunner, undo: noopUndoer}
			stages := []stage.Stage{mockStage1, mockStage2, errorStage}
			_, err := stage.RunStages(nullLogger, stages, 42)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("this is an error"))
			Expect(mockStage1.outputData).To(Equal(41))
			Expect(mockStage2.outputData).To(Equal(42))
		})

		It("outputs both the stage error and the rollback error if rollback failed", func() {
			buf := &bytes.Buffer{}
			logger := log.New(buf, "", log.LstdFlags)

			dummyRunner := func(data int) (int, error) {
				return data + 1, nil
			}
			errorRunner := func(data int) (int, error) {
				return 0, fmt.Errorf("this is an error")
			}
			dummyUndoer := func(data int) (int, error) {
				return 0, fmt.Errorf("this is also an error")
			}
			mockStage1 := &mockStage{run: dummyRunner, undo: dummyUndoer}
			mockStage2 := &mockStage{run: dummyRunner, undo: dummyUndoer}
			errorStage := &mockStage{run: errorRunner, undo: noopUndoer}
			stages := []stage.Stage{mockStage1, mockStage2, errorStage}
			_, err := stage.RunStages(logger, stages, 42)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("this is an error"))
			Expect(buf.String()).To(ContainSubstring("failed to roll back stages: this is also an error"))
		})
	})
})
