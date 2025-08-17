package scsi

func (d *SCSIDevice) TestUnitReady() (bool, error) {
	resp, err := d.request([]byte{
		TEST_UNIT_READY, 0x00, 0x00, 0x00, 0x00, 0x00,
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
