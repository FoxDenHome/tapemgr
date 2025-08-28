package loader

import (
	"github.com/FoxDenHome/tapemgr/scsi"
	"github.com/FoxDenHome/tapemgr/scsi/element"
)

func (l *TapeLoader) GetVolumeTags() ([]string, error) {
	dev, err := scsi.Open(l.DevicePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = dev.Close()
	}()

	elements, err := dev.ReadElementStatus(element.ELEMENT_TYPE_ALL, 0, LOADER_MAX_ELEMENTS, true, true, false)
	if err != nil {
		return nil, err
	}

	var barcodes []string
	for _, elem := range elements {
		if elem.HasFlag(element.FLAG_FULL) {
			barcodes = append(barcodes, elem.VolumeTag)
		}
	}

	return barcodes, nil
}
