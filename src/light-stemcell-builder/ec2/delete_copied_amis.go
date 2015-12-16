package ec2

import (
	"light-stemcell-builder/ec2/ec2ami"
)

func DeleteCopiedAmis(aws AWS, amiCollection *ec2ami.Collection) error {
	// for each region in destinations, launch a go routine that calls ec2cli.CopyImage and inserts success into amiCollection
	// on first error, cancel all other goroutines, and cleanup any created amis

	for _, amiInfo := range amiCollection.GetAll() {
		err := DeleteAmi(aws, amiInfo)
		if err != nil {
			return err
		}
	}
	return nil
}
