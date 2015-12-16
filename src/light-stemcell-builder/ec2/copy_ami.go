package ec2

import (
	"fmt"
	"light-stemcell-builder/ec2/ec2ami"
	"strings"
	"sync"
	"time"
)

func CopyAmis(aws AWS, amiInfo ec2ami.Info, destinations []string) (*ec2ami.Collection, error) {
	// for each region in destinations, launch a go routine that calls ec2cli.CopyImage and inserts success into amiCollection
	// on first error, cancel all other goroutines, and cleanup any created amis

	if validationError := amiInfo.InputConfig.Validate(); validationError != nil {
		return &ec2ami.Collection{}, validationError
	}

	wg := sync.WaitGroup{}
	amiCollection := ec2ami.NewCollection()
	errCollection := &errorCollection{}

	for i := range destinations {

		wg.Add(1)
		go func(dest string) {
			fmt.Printf("start go routine with dest %s\n", dest)
			defer wg.Done()
			amiInfo, err := copyAmi(aws, amiInfo, dest)
			if err != nil {
				fmt.Printf("failed copying to region %s, due to error [%s]. dirty state possible\n", dest, err.Error())
				errCollection.Add(err)
				return
			}
			if validationError := amiInfo.InputConfig.Validate(); validationError != nil {
				fmt.Printf("failed to validate copyAmi's configuration")
				errCollection.Add(validationError)
				return
			}

			amiCollection.Add(dest, amiInfo)
			fmt.Println("finished go routine")
		}(destinations[i])
	}

	wg.Wait()
	copyErrs := errCollection.GetAll()
	if len(copyErrs) > 0 {
		err := DeleteCopiedAmis(aws, amiCollection)
		if err != nil {
			return &ec2ami.Collection{}, fmt.Errorf(
				"failed to copy %s to all regions [%s]. failed to delete copied amis: %s",
				amiInfo.AmiID,
				strings.Join(destinations, ", "),
				err,
			)
		}
		return &ec2ami.Collection{}, fmt.Errorf(
			"failed to copy %s to all regions [%s]",
			amiInfo.AmiID,
			strings.Join(destinations, ", "),
		)

	}

	return amiCollection, nil
}

type errorCollection struct {
	errs []error
	sync.Mutex
}

func (c *errorCollection) Add(err error) {
	c.Lock()
	defer c.Unlock()

	c.errs = append(c.errs, err)
}

func (c *errorCollection) GetAll() []error {
	c.Lock()
	defer c.Unlock()

	return c.errs
}

func copyAmi(aws AWS, amiInfo ec2ami.Info, dest string) (ec2ami.Info, error) {
	copiedAmiID, err := aws.CopyImage(amiInfo.InputConfig, dest)
	if err != nil {
		return ec2ami.Info{}, err
	}

	copiedAmiConfig := amiInfo.InputConfig
	copiedAmiConfig.AmiID = copiedAmiID
	copiedAmiConfig.Region = dest

	waiterConfig := WaiterConfig{
		Resource:      &copiedAmiConfig,
		DesiredStatus: ec2ami.AmiAvailableStatus,
		PollTimeout:   15 * time.Minute,
	}

	fmt.Printf("waiting for status %s to %s\n", amiInfo.AmiID, copiedAmiID)
	statusInfo, err := WaitForStatus(aws.DescribeImage, waiterConfig)
	if err != nil {
		return ec2ami.Info{}, fmt.Errorf("waiting for copied ami %s to be available %s", copiedAmiID, err)
	}

	err = aws.MakeImagePublic(copiedAmiConfig)
	if err != nil {
		return ec2ami.Info{}, err
	}

	return statusInfo.(ec2ami.Info), nil
}
