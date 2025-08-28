package scsi

import (
	"time"

	"github.com/FoxDenHome/goscsi"
)

type SCSIDevice struct {
	dev goscsi.Dev
}

func Open(path string) (*SCSIDevice, error) {
	dev, err := goscsi.Open(path)
	if err != nil {
		return nil, err
	}
	return &SCSIDevice{dev: dev}, nil
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

func (d *SCSIDevice) requestWithTimeout(req []byte, respLen int, timeout time.Duration) ([]byte, error) {
	resp := make([]byte, respLen)
	err := d.dev.RequestWithTimeout(req, resp, timeout)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (d *SCSIDevice) SerialNumber() (string, error) {
	str, err := d.dev.SerialNumber()
	return str.String(), err
}
