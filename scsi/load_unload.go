package scsi

import "time"

type LoadUnloadOperation uint8

const (
	LOAD_AND_THREAD LoadUnloadOperation = 0b0001
	LOAD_ONLY       LoadUnloadOperation = 0b1001
	UNLOAD_FAST     LoadUnloadOperation = 0b0000
	UNLOAD_ARCHIVE  LoadUnloadOperation = 0b0010
)

func (d *SCSIDevice) LoadUnload(op LoadUnloadOperation) error {
	_, err := d.requestWithTimeout([]byte{
		LOAD_UNLOAD,
		0x00, // Lowest bit is immediate, we want to always wait
		0x00,
		0x00,
		uint8(op),
		0x00,
	}, 6, time.Minute*5)
	return err
}
