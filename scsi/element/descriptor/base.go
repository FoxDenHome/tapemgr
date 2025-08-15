package descriptor

import (
	"errors"
	"fmt"

	"github.com/FoxDenHome/tapemgr/scsi/element"
)

type Interface interface {
	Address() uint16
	ElementType() element.Type
}

type Base struct {
	address     uint16
	elementType element.Type
}

type descriptorConstruct func(data []byte, hasVolTag bool, base *Base) (Interface, error)

var descriptorConstructors = map[element.Type]descriptorConstruct{
	element.ELEMENT_TYPE_MEDIUM_TRANSPORT: parseMediumTransport,
	element.ELEMENT_TYPE_STORAGE:          parseStorage,
	element.ELEMENT_TYPE_IMPORT_EXPORT:    parseImportExport,
	element.ELEMENT_TYPE_DATA_TRANSFER:    parseDataTransfer,
}

func Parse(elementType element.Type, hasVolTag bool, data []byte) (Interface, error) {
	if len(data) < 2 {
		return nil, errors.New("invalid data length")
	}

	baseDesc := &Base{
		address:     uint16(data[0])<<8 | uint16(data[1]),
		elementType: elementType,
	}

	if constructor, ok := descriptorConstructors[elementType]; ok {
		return constructor(data, hasVolTag, baseDesc)
	}

	return nil, fmt.Errorf("unsupported element type %v", elementType)
}

func (e *Base) Address() uint16 {
	return e.address
}

func (e *Base) ElementType() element.Type {
	return e.elementType
}
