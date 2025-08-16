package scsi

const (
	TEST_UNIT_READY     = 0x00
	LOAD_UNLOAD         = 0x1B
	POSITION_TO_ELEMENT = 0x2B
	MOVE_MEDIUM         = 0xA5
	READ_ELEMENT_STATUS = 0xB8
)
