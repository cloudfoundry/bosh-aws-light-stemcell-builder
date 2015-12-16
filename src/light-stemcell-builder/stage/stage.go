package stage

import (
	"fmt"
	"log"
)

// A Stage represents a collection of calls whose effects can be reverted by calling
// Rollback(). A Stage must clean up any intermediate resources created in Run()
// Each Stage must know the data type expected in Run()
type Stage interface {
	Run(logger *log.Logger, data interface{}) (interface{}, error)
	Rollback(logger *log.Logger) error
	Name() string
}

// EmptyData can be used for Run() invocations which do not require input data
var EmptyData = struct{}{}

// RunStages takes in preconfigured stages and input data and returns the outputs of each
// of those stages. These stages are run in order, passing in one stage's output to the
// next stage.
func RunStages(logger *log.Logger, stages []Stage, inputData interface{}) ([]interface{}, error) {
	var completedStages []Stage
	returnData := []interface{}{}
	data := inputData
	var err error
	for _, stage := range stages {
		logger.SetPrefix(fmt.Sprintf("%s: ", stage.Name()))
		data, err = stage.Run(logger, data)
		returnData = append(returnData, data)
		if err != nil {
			rollbackErr := rollbackStages(logger, completedStages)
			if rollbackErr != nil {
				logger.Printf("failed to roll back stages: %s\n", rollbackErr)
			}
			return nil, err
		}
		completedStages = append(completedStages, stage)
	}
	return returnData, nil
}

func rollbackStages(logger *log.Logger, stages []Stage) error {
	for _, stage := range stages {
		err := stage.Rollback(logger)
		if err != nil {
			return err
		}
	}
	return nil
}
