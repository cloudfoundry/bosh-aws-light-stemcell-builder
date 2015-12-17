package ec2stage

import (
	"fmt"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/stage"
	"log"
	"reflect"
)

type CreateEBSVolumeRunner func(aws ec2.AWS, imagePath string) (ec2.ConversionTaskInfo, error)
type CreateEBSVolumeCleaner func(aws ec2.AWS, taskID string) error
type CreateEBSVolumeUndoer func(aws ec2.AWS, volumeID string) error

type createEBSVolumeStage struct {
	aws      ec2.AWS
	run      CreateEBSVolumeRunner
	clean    CreateEBSVolumeCleaner
	undo     CreateEBSVolumeUndoer
	volumeID string
}

func NewCreateEBSVolumeStage(runner CreateEBSVolumeRunner, cleaner CreateEBSVolumeCleaner, undoer CreateEBSVolumeUndoer, aws ec2.AWS) stage.Stage {
	return &createEBSVolumeStage{
		run:   runner,
		clean: cleaner,
		undo:  undoer,
		aws:   aws,
	}
}

func (s *createEBSVolumeStage) Run(logger *log.Logger, data interface{}) (interface{}, error) {
	if reflect.TypeOf(data) != reflect.TypeOf("") {
		return nil, fmt.Errorf("CreateVolume expected type string, got: %s", reflect.TypeOf(data))
	}

	logger.Printf("Running stage with data: %s\n", data.(string))
	taskInfo, err := s.run(s.aws, data.(string))
	if err != nil {
		return nil, fmt.Errorf("CreateVolume error running: %s", err)
	}
	logger.Println("Successfully finished running stage.")

	logger.Printf("Cleaning up stage with task ID: %s\n", taskInfo.TaskID)
	err = s.clean(s.aws, taskInfo.TaskID)
	if err != nil {
		return nil, fmt.Errorf("CreateVolume error cleaning after run: %s", err)
	}

	s.volumeID = taskInfo.EBSVolumeID
	logger.Printf("Output of stage: %s\n", s.volumeID)
	return s.volumeID, nil
}

func (s *createEBSVolumeStage) Rollback(logger *log.Logger) error {
	logger.Printf("Rolling back stage with input: %s\n", s.volumeID)
	return s.undo(s.aws, s.volumeID)
}

func (s *createEBSVolumeStage) Name() string {
	return "CreateEBSVolume"
}
