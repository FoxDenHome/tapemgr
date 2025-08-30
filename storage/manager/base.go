package manager

import (
	"fmt"

	"github.com/FoxDenHome/tapemgr/scsi/drive"
	"github.com/FoxDenHome/tapemgr/scsi/loader"
	"github.com/FoxDenHome/tapemgr/storage/encryption"
	"github.com/FoxDenHome/tapemgr/storage/inventory"
)

var DryRun = true

type Manager struct {
	file *encryption.FileCryptor
	path *encryption.PathCryptor

	inventory          *inventory.Inventory
	loader             *loader.TapeLoader
	drive              *drive.TapeDrive
	loaderDriveAddress uint16

	currentTape *inventory.Tape
}

func New(
	file *encryption.FileCryptor,
	path *encryption.PathCryptor,
	inventory *inventory.Inventory,
	loader *loader.TapeLoader,
	drive *drive.TapeDrive,
) (*Manager, error) {
	serialNumber, err := drive.SerialNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to get tape drive serial number: %v", err)
	}

	address, err := loader.DriveAddressBySerial(serialNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get tape drive address: %v", err)
	}

	return &Manager{
		file: file,
		path: path,

		inventory:          inventory,
		loader:             loader,
		drive:              drive,
		loaderDriveAddress: address,
	}, nil
}
