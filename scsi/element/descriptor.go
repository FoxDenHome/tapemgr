package element

import (
	"bytes"
	"fmt"
)

const (
	VolumeTagLength   = 36
	DeviceIDLengthMax = 32
)

type MediumType uint8
type CodeSet uint8
type IdentifierType uint8
type Flag uint16

const (
	IDENTIFIER_TYPE_VENDOR IdentifierType = 0x00

	CODE_SET_UNDEFINED CodeSet = 0x00
	CODE_SET_ASCII     CodeSet = 0x02

	FLAG_FULL           Flag = 1 << 0
	FLAG_IMPORT_EXPORT  Flag = 1 << 1
	FLAG_EXCEPTION      Flag = 1 << 2
	FLAG_ACCESS         Flag = 1 << 3
	FLAG_EXPORT_ENABLED Flag = 1 << 4
	FLAG_IMPORT_ENABLED Flag = 1 << 5
	FLAG_CMC            Flag = 1 << 6
	FLAG_OIR            Flag = 1 << 7

	FLAG_ED                  Flag = 1 << (8 + 3)
	FLAG_INVERT              Flag = 1 << (8 + 6)
	FLAG_SOURCE_INVERT_VALID Flag = 1 << (8 + 7)
)

type Descriptor struct {
	Address                     uint16
	ElementType                 Type
	Flags                       uint16
	ExceptionSenseCode          uint8
	ExceptionSenseCodeQualifier uint8
	MediumType                  MediumType
	SourceElementAddress        uint16
	CodeSet                     CodeSet
	IdentifierType              IdentifierType
	VolumeTag                   string
	Identifier                  string
}

func ParseDescriptor(elementType Type, hasPVolTag bool, data []byte) (*Descriptor, error) {
	dataLength := 16
	if hasPVolTag {
		dataLength += VolumeTagLength
	}
	if len(data) < dataLength {
		return nil, fmt.Errorf("too small data length for element descriptor: expected >= %d, got %d", dataLength, len(data))
	}

	volTagEnd := 12
	if hasPVolTag {
		volTagEnd += VolumeTagLength
	}
	identifierLen := int(data[volTagEnd+3])

	baseDesc := &Descriptor{
		Address:     uint16(data[0])<<8 | uint16(data[1]),
		ElementType: elementType,

		ExceptionSenseCode:          data[4],
		ExceptionSenseCodeQualifier: data[5],
		MediumType:                  MediumType(data[9] & 0b111),
		SourceElementAddress:        uint16(data[10])<<8 | uint16(data[11]),
		CodeSet:                     CodeSet(data[volTagEnd] & 0x0F),
		IdentifierType:              IdentifierType(data[volTagEnd+1] & 0x0F),
		Identifier:                  string(bytes.Trim(data[volTagEnd+4:volTagEnd+4+identifierLen], "\x00 ")),

		Flags: uint16(data[9])<<8 | uint16(data[2]),
	}

	if hasPVolTag {
		baseDesc.VolumeTag = string(bytes.Trim(data[12:48], "\x00 "))
	}

	return baseDesc, nil
}

func (d *Descriptor) HasFlag(flag Flag) bool {
	return d.Flags&uint16(flag) != 0
}
