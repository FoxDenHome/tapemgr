package loader

import (
	"fmt"
	"log"

	"github.com/FoxDenHome/tapemgr/scsi"
	"github.com/FoxDenHome/tapemgr/scsi/element"
)

type TapeLoader struct {
	DevicePath string
}

func NewTapeLoader(devicePath string) (*TapeLoader, error) {
	return &TapeLoader{
		DevicePath: devicePath,
	}, nil
}

func (l *TapeLoader) DriveAddressBySerial(serial string) (uint16, error) {
	dev, err := scsi.Open(l.DevicePath)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = dev.Close()
	}()

	elements, err := dev.ReadElementStatus(element.ELEMENT_TYPE_DATA_TRANSFER, 0, 255, true, false, true)
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

func (l *TapeLoader) MoveTapeToDrive(driveAddress uint16, volumeTag string) error {
	dev, err := scsi.Open(l.DevicePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = dev.Close()
	}()

	elements, err := dev.ReadElementStatus(element.ELEMENT_TYPE_ALL, 0, 255, true, true, false)
	if err != nil {
		return err
	}

	for _, elem := range elements {
		if elem.VolumeTag == volumeTag {
			if elem.Address == driveAddress {
				return nil
			}
			log.Printf("moving tape %s from address %d to drive %d", volumeTag, elem.Address, driveAddress)
			return dev.MoveMedium(elem.Address, driveAddress, scsi.MOVE_OPTION_NORMAL)
		}
	}

	return fmt.Errorf("no tape found with volume tag %s", volumeTag)
}

func (l *TapeLoader) MoveDriveTapeToStorage(driveAddress uint16) error {
	dev, err := scsi.Open(l.DevicePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = dev.Close()
	}()

	elements, err := dev.ReadElementStatus(element.ELEMENT_TYPE_DATA_TRANSFER, driveAddress, 1, true, false, false)
	if err != nil {
		return err
	}

	if len(elements) != 1 {
		return fmt.Errorf("expected 1 data transfer element, got %d", len(elements))
	}

	sourceAddr := elements[0].SourceElementAddress
	if sourceAddr != 0 {
		elements, err = dev.ReadElementStatus(element.ELEMENT_TYPE_STORAGE, sourceAddr, 1, true, false, false)
		if err != nil {
			return err
		}

		if len(elements) == 1 && !elements[0].HasFlag(element.FLAG_FULL) {
			log.Printf("Moving tape from drive %d back to source %d", driveAddress, sourceAddr)
			return dev.MoveMedium(driveAddress, sourceAddr, scsi.MOVE_OPTION_NORMAL)
		}
	}

	elements, err = dev.ReadElementStatus(element.ELEMENT_TYPE_STORAGE, 0, 255, true, false, false)
	if err != nil {
		return err
	}

	for _, elem := range elements {
		if elem.HasFlag(element.FLAG_FULL) {
			continue
		}
		log.Printf("Moving tape from drive %d to free slot %d", driveAddress, elem.Address)
		return dev.MoveMedium(driveAddress, elem.Address, scsi.MOVE_OPTION_NORMAL)
	}

	return fmt.Errorf("no free slot found for tape in drive %d", driveAddress)
}
