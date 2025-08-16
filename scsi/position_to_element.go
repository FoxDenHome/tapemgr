package scsi

func (d *SCSIDevice) PositionToElement(destAddress uint16) error {
	_, err := d.request([]byte{
		POSITION_TO_ELEMENT,
		0x00,
		0x00, 0x00, // Transport element address, no library seems to care about this and auto-select the arm instead
		uint8(destAddress >> 8), uint8(destAddress & 0xFF),
		0x00,
		0x00, // Last bit is invert flag, but this is not supported
		0x00, // Control byte, always 0
	}, 0)

	return err
}
