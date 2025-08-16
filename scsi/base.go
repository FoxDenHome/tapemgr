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

func (d *SCSIDevice) Close() error {
	return d.dev.Close()
}

func (d *SCSIDevice) request(req []byte, respLen int) ([]byte, error) {
	resp := make([]byte, respLen)
	err := d.dev.Request(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
