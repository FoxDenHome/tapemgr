package scsi

import (
	"errors"

	scsidefs "github.com/FoxDenHome/goscsi/godefs/scsi"
	"github.com/FoxDenHome/tapemgr/scsi/element"
)

func (d *SCSIDevice) ReadElementStatus(elementType element.Type, start uint16, count uint16, curData bool, readVolumeTag bool, readDeviceId bool) ([]*element.Descriptor, error) {
	perElementLen := 52 // Maximum plain response len observed in docs
	if readVolumeTag {
		perElementLen += element.VolumeTagLength
	}
	if readDeviceId {
		perElementLen += element.DeviceIDLengthMax
	}
	// Expected element length + (page header * max pages [one per element type at most]) + header
	reservedRespLen := (perElementLen * int(count)) + (8 * element.ElementTypes) + 8

	resp, err := d.request([]byte{
		scsidefs.READ_ELEMENT_STATUS,
		boolToFlag(readVolumeTag, 4) | uint8(elementType),
		uint8(start >> 8), uint8(start & 0xFF),
		uint8(count >> 8), uint8(count & 0xFF),
		boolToFlag(curData, 1) | boolToFlag(readDeviceId, 0),
		uint8(reservedRespLen >> 16), uint8((reservedRespLen >> 8) & 0xFF), uint8(reservedRespLen & 0xFF),
		0x00, 0x00,
	}, reservedRespLen)
	if err != nil {
		return nil, err
	}

	var elementStatuses []*element.Descriptor
	reportLength := int(resp[5])<<16 | int(resp[6])<<8 | int(resp[7])

	pos := 8
	for pos < reportLength+8 {
		elementType := element.Type(resp[pos] & 0x0F)
		elementLength := int(resp[pos+2])<<8 | int(resp[pos+3])
		pageLength := int(resp[pos+5])<<16 | int(resp[pos+6])<<8 | int(resp[pos+7])

		hasPVolTag := flagToBool(resp[pos+1], 7)
		hasAVolTag := flagToBool(resp[pos+1], 6)

		if hasAVolTag {
			return nil, errors.New("received Alternative Volume Tag, which is not supported")
		}

		pos += 8
		subPos := 0
		for subPos < pageLength && pos+subPos+elementLength <= reportLength+8 {
			desc, err := element.ParseDescriptor(elementType, hasPVolTag, resp[pos+subPos:pos+subPos+elementLength])
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
