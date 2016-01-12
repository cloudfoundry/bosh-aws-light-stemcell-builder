package ec2stage

import (
	"light-stemcell-builder/ec2"
	"log"
)

// NullCleaner fakes cleaning. Due to a bug in the ec2-cli we cannot clean
// unneeded machine images from S3 when the bucket is located in China. For now, fake cleaning
func NullCleaner(aws ec2.AWS, discardedID string) error {
	log.Printf("Refusing to clean resource with ID: %s\n please clean manually", discardedID)
	return nil
}
