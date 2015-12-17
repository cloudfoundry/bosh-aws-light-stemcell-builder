package ec2stage

import (
	"fmt"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2ami"
	"light-stemcell-builder/stage"
	"log"
	"reflect"
)

type CreateAmiRunner func(aws ec2.AWS, volumeID string, amiConfig ec2ami.Config) (ec2ami.Info, error)
type CreateAmiUndoer func(aws ec2.AWS, amiInfo ec2ami.Info) error

type createAmiStage struct {
	aws       ec2.AWS
	amiConfig ec2ami.Config
	run       CreateAmiRunner
	undo      CreateAmiUndoer
	amiInfo   ec2ami.Info
}

func NewCreateAmiStage(runner CreateAmiRunner, undoer CreateAmiUndoer, aws ec2.AWS, amiConfig ec2ami.Config) stage.Stage {
	return &createAmiStage{
		run:       runner,
		undo:      undoer,
		amiConfig: amiConfig,
		aws:       aws,
	}
}

func (s *createAmiStage) Run(logger *log.Logger, data interface{}) (interface{}, error) {
	if reflect.TypeOf(data) != reflect.TypeOf("") {
		return nil, fmt.Errorf("CreateAmi expected type string, got: %s", reflect.TypeOf(data))
	}

	logger.Printf("Running stage with data: %s\n", data.(string))
	var err error
	s.amiInfo, err = s.run(s.aws, data.(string), s.amiConfig)
	if err != nil {
		return nil, fmt.Errorf("CreateAmi error running: %s", err)
	}

	logger.Printf("Output of stage : %s\n", s.amiInfo)
	return s.amiInfo, nil
}

func (s *createAmiStage) Rollback(logger *log.Logger) error {
	logger.Printf("Rolling back stage with input: %s\n", s.amiInfo)
	return s.undo(s.aws, s.amiInfo)
}

func (s *createAmiStage) Name() string {
	return "CreateAmi"
}
