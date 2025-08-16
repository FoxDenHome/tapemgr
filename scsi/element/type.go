package element

type Type uint8

const (
	ELEMENT_TYPE_ALL              Type = 0x00
	ELEMENT_TYPE_MEDIUM_TRANSPORT Type = 0x01
	ELEMENT_TYPE_STORAGE          Type = 0x02
	ELEMENT_TYPE_IMPORT_EXPORT    Type = 0x03
	ELEMENT_TYPE_DATA_TRANSFER    Type = 0x04
)

const ElementTypes = 4

func (t Type) String() string {
	switch t {
	case ELEMENT_TYPE_ALL:
		return "All"
	case ELEMENT_TYPE_MEDIUM_TRANSPORT:
		return "Medium Transport"
	case ELEMENT_TYPE_STORAGE:
		return "Storage"
	case ELEMENT_TYPE_IMPORT_EXPORT:
		return "Import/Export"
	case ELEMENT_TYPE_DATA_TRANSFER:
		return "Data Transfer"
	default:
		return "Unknown"
	}
}
