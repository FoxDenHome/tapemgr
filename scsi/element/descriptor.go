package element

import (
	"fmt"
)

const VOLUME_TAG_LENGTH = 36

type MediumType uint8
type CodeSet uint8
type IdentifierType uint8

const (
	IDENTIFIER_TYPE_VENDOR IdentifierType = 0x00

	CODE_SET_UNDEFINED CodeSet = 0x00
	CODE_SET_ASCII     CodeSet = 0x02

	FLAG_FULL           = 1 << 0
	FLAG_IMPORT_EXPORT  = 1 << 1
	FLAG_EXCEPTION      = 1 << 2
	FLAG_ACCESS         = 1 << 3
	FLAG_EXPORT_ENABLED = 1 << 4
	FLAG_IMPORT_ENABLED = 1 << 5
	FLAG_CMC            = 1 << 6
	FLAG_OIR            = 1 << 7

	FLAG_ED                  = 1 << (8 + 3)
	FLAG_INVERT              = 1 << (8 + 6)
	FLAG_SOURCE_INVERT_VALID = 1 << (8 + 7)
)

type Descriptor struct {
	Address     uint16
	ElementType Type

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

func ParseDescriptor(elementType Type, hasVolTag bool, data []byte) (*Descriptor, error) {
	dataLength := 16
	if hasVolTag {
		dataLength += VOLUME_TAG_LENGTH
	}
	if len(data) < dataLength {
		return nil, fmt.Errorf("too small data length for element descriptor: expected >= %d, got %d", dataLength, len(data))
	}

	volTagEnd := 12
	if hasVolTag {
		volTagEnd += VOLUME_TAG_LENGTH
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
		Identifier:                  string(data[volTagEnd+4 : volTagEnd+4+identifierLen]),

		Flags: uint16(data[9])<<8 | uint16(data[2]),
	}

	if hasVolTag {
		baseDesc.VolumeTag = string(data[12:48])
	}

	return baseDesc, nil
}
