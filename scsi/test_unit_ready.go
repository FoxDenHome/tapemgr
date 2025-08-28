package scsi

import (
	scsidefs "github.com/FoxDenHome/goscsi/godefs/scsi"
)

func (d *SCSIDevice) TestUnitReady() (bool, error) {
	resp, err := d.request([]byte{
		scsidefs.TEST_UNIT_READY, 0x00, 0x00, 0x00, 0x00, 0x00,
	}, 6)
	if err != nil {
		return false, err
	}

	return resp[5] == 0x00, nil
}

func (d *SCSIDevice) WaitForReady() error {
	for {
		ready, err := d.TestUnitReady()
		if err != nil {
			return err
		}
		if ready {
			return nil
		}
	}
}
