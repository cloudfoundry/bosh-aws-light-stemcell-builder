package builder

import (
	"light-stemcell-builder/config"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2ami"
	"light-stemcell-builder/ec2/ec2stage"
	"light-stemcell-builder/stage"
	"log"
)

type AMIPublisher struct {
	AWS       ec2.AWS
	AMIConfig config.AmiConfiguration
	Logger    *log.Logger
}

func (p *AMIPublisher) Publish(imagePath string, region config.RegionConfiguration) (map[string]ec2ami.Info, error) {
	ec2Config := ec2.Config{
		Region:     region.Name,
		BucketName: region.BucketName,
		Credentials: &ec2.Credentials{
			AccessKey: region.Credentials.AccessKey,
			SecretKey: region.Credentials.SecretKey,
		},
	}
	amiConfig := ec2ami.Config{
		Description:        p.AMIConfig.Description,
		VirtualizationType: p.AMIConfig.VirtualizationType,
		Public:             p.AMIConfig.Visibility == "public",
		Region:             region.Name,
	}

	p.AWS.Configure(ec2Config)

	ebsVolumeStage := ec2stage.NewCreateEBSVolumeStage(ec2.ImportVolume,
		ec2.CleanupImportVolume, ec2.DeleteVolume, p.AWS)

	createAmiStage := ec2stage.NewCreateAmiStage(ec2.CreateAmi, ec2.DeleteAmi, p.AWS, amiConfig)

	copyAmiStage := ec2stage.NewCopyAmiStage(ec2.CopyAmis, ec2.DeleteCopiedAmis, p.AWS, region.Destinations)

	outputData, err := stage.RunStages(p.Logger, []stage.Stage{ebsVolumeStage, createAmiStage, copyAmiStage}, imagePath)
	if err != nil {
		return nil, err
	}

	// Delete the EBS Volume created for the AMI; We no longer need it at this point
	err = ec2.DeleteVolume(p.AWS, outputData[0].(string))
	if err != nil {
		p.Logger.Printf("Unable to clean up volume due to error: %s\n", err.Error())
	}

	copiedAmiCollection := outputData[2].(*ec2ami.Collection)
	resultMap := copiedAmiCollection.GetAll()
	resultMap[region.Name] = outputData[1].(ec2ami.Info)

	return resultMap, nil
}
