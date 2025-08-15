package scsi

import "github.com/platinasystems/scsi"

type SCSIDevice struct {
	dev *scsi.Dev
}

func NewSCSIDevice(path string) (*SCSIDevice, error) {
	dev, err := scsi.Open(path)
	if err != nil {
		return nil, err
	}
	return &SCSIDevice{dev: &dev}, nil
}
