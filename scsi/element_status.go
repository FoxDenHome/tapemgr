package scsi

import (
	"github.com/FoxDenHome/tapemgr/scsi/element"
	"github.com/FoxDenHome/tapemgr/scsi/element/descriptor"
)

const (
	READ_ELEMENT_STATUS = 0xB8
)

func boolToFlag(val bool, pos uint8) uint8 {
	if val {
		return 1 << pos
	}
	return 0
}

func (d *SCSIDevice) ReadElementStatus(lun uint8, elementType element.Type, start uint16, count uint16, curData bool, readVolumeTag bool, readDeviceId bool) ([]descriptor.Interface, error) {
	respLen := 65536
	resp := make([]byte, respLen)

	req := []byte{
		READ_ELEMENT_STATUS,
		lun<<5 | boolToFlag(readVolumeTag, 4) | uint8(elementType),
		uint8(start >> 8), uint8(start & 0xFF),
		uint8(count >> 8), uint8(count & 0xFF),
		boolToFlag(curData, 1) | boolToFlag(readDeviceId, 0),
		uint8(respLen >> 16), uint8((respLen >> 8) & 0xFF), uint8(respLen & 0xFF),
		0x00, 0x00,
	}

	err := d.dev.Request(req, resp)
	if err != nil {
		return nil, err
	}

	var elementStatuses []descriptor.Interface
	/// address := uint16(resp[0])<<8 | uint16(resp[1])
	// elementCount := uint16(resp[2])<<8 | uint16(resp[3])
	reportLength := int(resp[5])<<16 | int(resp[6])<<8 | int(resp[7])

	pos := 8
	for pos < reportLength+8 {
		elementType := element.Type(resp[pos] & 0x0F)
		elementLength := int(resp[pos+2])<<8 | int(resp[pos+3])
		pageLength := int(resp[pos+5])<<16 | int(resp[pos+6])<<8 | int(resp[pos+7])

		hasVolTag := resp[pos+1]&(1<<7) != 0

		pos += 8
		subPos := 0
		for subPos < pageLength && pos+subPos+elementLength <= reportLength+8 {
			desc, err := descriptor.Parse(elementType, hasVolTag, resp[pos+subPos:pos+subPos+elementLength])
			if err != nil {
				return nil, err
			}
			elementStatuses = append(elementStatuses, desc)
			subPos += elementLength
		}

		pos += pageLength
	}

	return elementStatuses, nil
}
