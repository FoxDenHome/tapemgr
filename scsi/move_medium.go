package scsi

import (
	"time"

	scsidefs "github.com/FoxDenHome/goscsi/godefs/scsi"
)

type MoveOption uint8

const (
	MOVE_OPTION_NORMAL          MoveOption = 0b00 << 6
	MOVE_OPTION_WRITE_PROTECTED MoveOption = 0b10 << 6
	MOVE_OPTION_REWIND_UNLOAD   MoveOption = 0b11 << 6
)

func (d *SCSIDevice) MoveMedium(sourceAddress uint16, destAddress uint16, moveOption MoveOption) error {
	_, err := d.requestWithTimeout([]byte{
		scsidefs.MOVE_MEDIUM,
		0x00,
		0x00, 0x00, // Transport element address, no library seems to care about this and auto-select the arm instead
		uint8(sourceAddress >> 8), uint8(sourceAddress & 0xFF),
		uint8(destAddress >> 8), uint8(destAddress & 0xFF),
		0x00,
		0x00,             // Last bit is invert flag, but this is not supported
		byte(moveOption), // Last 5 bits are control byte, which are always 0
	}, 6, time.Minute*5)
	return err
}
