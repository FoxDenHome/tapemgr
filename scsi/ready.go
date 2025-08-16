package scsi

func (d *SCSIDevice) IsReady() (bool, error) {
	resp, err := d.request([]byte{
		TEST_UNIT_READY, 0x00, 0x00, 0x00, 0x00, 0x00,
	}, 6)
	if err != nil {
		return false, err
	}

	return resp[5] == 0x00, nil
}
