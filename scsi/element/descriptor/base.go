package descriptor

import (
	"fmt"

	"github.com/FoxDenHome/tapemgr/scsi/element"
	"github.com/FoxDenHome/tapemgr/util"
)

const VOLUME_TAG_LENGTH = 36

type Interface interface {
	Address() uint16
	ElementType() element.Type
}

type MediumType uint8
type CodeSet uint8
type IdentifierType uint8

const (
	IDENTIFIER_TYPE_VENDOR IdentifierType = 0x00

	CODE_SET_UNDEFINED CodeSet = 0x00
	CODE_SET_ASCII     CodeSet = 0x02
)

type Base struct {
	address     uint16
	elementType element.Type

	SenseCode            uint8
	SenseCodeQualifier   uint8
	SValid               bool
	Invert               bool
	ED                   bool
	MediumType           MediumType
	SourceElementAddress uint16
	CodeSet              CodeSet
	IdentifierType       IdentifierType
	VolumeTag            string
	Identifier           string
}

type descriptorConstruct func(data []byte, base *Base) (Interface, error)

var descriptorConstructors = map[element.Type]descriptorConstruct{
	element.ELEMENT_TYPE_MEDIUM_TRANSPORT: parseMediumTransport,
	element.ELEMENT_TYPE_STORAGE:          parseStorage,
	element.ELEMENT_TYPE_IMPORT_EXPORT:    parseImportExport,
	element.ELEMENT_TYPE_DATA_TRANSFER:    parseDataTransfer,
}

func Parse(elementType element.Type, hasVolTag bool, data []byte) (Interface, error) {
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

	baseDesc := &Base{
		address:     uint16(data[0])<<8 | uint16(data[1]),
		elementType: elementType,

		SenseCode:            data[4],
		SenseCodeQualifier:   data[5],
		SValid:               util.FlagToBool(data[9], 7),
		Invert:               util.FlagToBool(data[9], 6),
		ED:                   util.FlagToBool(data[9], 3),
		MediumType:           MediumType(data[9] & 0b111),
		SourceElementAddress: uint16(data[10])<<8 | uint16(data[11]),
		CodeSet:              CodeSet(data[volTagEnd] & 0x0F),
		IdentifierType:       IdentifierType(data[volTagEnd+1] & 0x0F),
		Identifier:           string(data[volTagEnd+4 : volTagEnd+4+identifierLen]),
	}

	if hasVolTag {
		baseDesc.VolumeTag = string(data[12:48])
	}

	if constructor, ok := descriptorConstructors[elementType]; ok {
		return constructor(data, baseDesc)
	}

	return nil, fmt.Errorf("unsupported element type %v", elementType)
}

func (e *Base) Address() uint16 {
	return e.address
}

func (e *Base) ElementType() element.Type {
	return e.elementType
}
