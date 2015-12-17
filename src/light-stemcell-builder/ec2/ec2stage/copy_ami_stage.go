package ec2stage

import (
	"fmt"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2ami"
	"light-stemcell-builder/stage"
	"reflect"
	"log"
)

type CopyAmiRunner func(aws ec2.AWS, amiInfo ec2ami.Info, destinations []string) (*ec2ami.Collection, error)
type CopyAmiUndoer func(aws ec2.AWS, amiCollection *ec2ami.Collection) error

type copyAmiStage struct {
	aws           ec2.AWS
	run           CopyAmiRunner
	undo          CopyAmiUndoer
	destinations  []string
	amiCollection *ec2ami.Collection
}

func NewCopyAmiStage(runner CopyAmiRunner, undoer CopyAmiUndoer, aws ec2.AWS, dest []string) stage.Stage {
	return &copyAmiStage{
		run:          runner,
		undo:         undoer,
		aws:          aws,
		destinations: dest,
	}
}

func (s *copyAmiStage) Name() string {
	return "CopyAmi"
}

func (s *copyAmiStage) Run(logger *log.Logger, data interface{}) (interface{}, error) {
	if reflect.TypeOf(data) != reflect.TypeOf(ec2ami.Info{}) {
		return nil, fmt.Errorf("CopyAmi expected type ec2ami.Info, got: %s", reflect.TypeOf(data))
	}

	logger.Printf("Running stage with data: %s\n", data.(ec2ami.Info))
	var err error
	s.amiCollection, err = s.run(s.aws, data.(ec2ami.Info), s.destinations)
	if err != nil {
		return nil, fmt.Errorf("CopyAmi error running: %s", err)
	}

	logger.Printf("Output of stage : %s\n", s.amiCollection)
	return s.amiCollection, nil
}

func (s *copyAmiStage) Rollback(logger *log.Logger) error {
	logger.Printf("Rolling back stage with data: %s\n", s.amiCollection)
	return s.undo(s.aws, s.amiCollection)
}
