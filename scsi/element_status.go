package scsi

func (d *SCSIDevice) ReadElementStatus() {
	d.dev.Request()
}
