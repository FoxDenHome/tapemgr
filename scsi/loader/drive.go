package loader

import (
	"fmt"

	"github.com/FoxDenHome/tapemgr/scsi"
	"github.com/FoxDenHome/tapemgr/scsi/element"
)

func (l *TapeLoader) DriveAddressBySerial(serial string) (uint16, error) {
	dev, err := scsi.Open(l.DevicePath)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = dev.Close()
	}()

	elements, err := dev.ReadElementStatus(element.ELEMENT_TYPE_DATA_TRANSFER, 0, LOADER_MAX_ELEMENTS, true, false, true)
	if err != nil {
		return 0, err
	}

	for _, elem := range elements {
		if serial == elem.Identifier[24:] {
			return elem.Address, nil
		}
	}

	return 0, fmt.Errorf("no drive with serial %s found in loader", serial)
}
