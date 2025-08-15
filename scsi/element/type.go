package element

type Type uint8

const (
	ELEMENT_TYPE_ALL              Type = 0x00
	ELEMENT_TYPE_MEDIUM_TRANSPORT Type = 0x01
	ELEMENT_TYPE_STORAGE          Type = 0x02
	ELEMENT_TYPE_IMPORT_EXPORT    Type = 0x03
	ELEMENT_TYPE_DATA_TRANSFER    Type = 0x04
)
