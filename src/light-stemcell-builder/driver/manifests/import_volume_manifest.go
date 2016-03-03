package manifests

import (
	"encoding/xml"
	"math"
)

const (
	importerName    = "aws-light-stemcell-builder"
	importerVersion = "0.1.0"
	importerRelease = "beta"
	fileFormat      = "RAW"
	awsAPIVersion   = "2010-11-15"
)

const gbInBytes = 1 << 30

// MachineImageProperties contains information needed by AWS to download a machine image from S3
type MachineImageProperties struct {
	KeyName   string
	HeadURL   string
	GetURL    string
	DeleteURL string
	SizeBytes int64
}

// ImportVolumeManifest will produce an Import Volume Manifest when marshalled to XML
type ImportVolumeManifest struct {
	XMLName         xml.Name        `xml:"manifest"`
	Version         string          `xml:"version"`
	FileFormat      string          `xml:"file-format"`
	ImporterName    string          `xml:"importer>name"`
	ImporterVersion string          `xml:"importer>version"`
	ImporterRelease string          `xml:"importer>release"`
	SelfDestructURL string          `xml:"self-destruct-url"`
	SizeBytes       int64           `xml:"import>size"`
	VolumeSizeGB    int64           `xml:"import>volume-size"`
	Parts           PartsCollection `xml:"import>parts"`
}

// PartsCollection is used for XML generation of a Import Volume Manifest
type PartsCollection struct {
	Count int              `xml:"count,attr"`
	Part  MachineImagePart `xml:"part"`
}

// MachineImagePart is used for XML generation of a Import Volume Manifest
type MachineImagePart struct {
	Index     int       `xml:"index,attr"`
	ByteRange ByteRange `xml:"byte-range"`
	Key       string    `xml:"key"`
	HeadURL   string    `xml:"head-url"`
	GetURL    string    `xml:"get-url"`
	DeleteURL string    `xml:"delete-url"`
}

// ByteRange is used for XML generation of a Import Volume Manifest
type ByteRange struct {
	Start int64 `xml:"start,attr"`
	End   int64 `xml:"end,attr"`
}

// New returns an Import Volume Manifest ready for marshalling to XML
func New(imageProperties MachineImageProperties, deleteManifestURL string) ImportVolumeManifest {
	roundedSizeGB := int64(math.Ceil(float64(imageProperties.SizeBytes) / gbInBytes))

	m := ImportVolumeManifest{
		SizeBytes:       imageProperties.SizeBytes,
		ImporterVersion: importerVersion,
		ImporterRelease: importerRelease,
		ImporterName:    importerName,
		Version:         awsAPIVersion,
		FileFormat:      fileFormat,
		SelfDestructURL: deleteManifestURL,
		VolumeSizeGB:    roundedSizeGB,
	}

	m.Parts.Count = 1
	m.Parts.Part.Index = 0
	m.Parts.Part.ByteRange.Start = 0
	m.Parts.Part.ByteRange.End = imageProperties.SizeBytes
	m.Parts.Part.Key = imageProperties.KeyName
	m.Parts.Part.GetURL = imageProperties.GetURL
	m.Parts.Part.HeadURL = imageProperties.HeadURL
	m.Parts.Part.DeleteURL = imageProperties.DeleteURL

	return m
}
