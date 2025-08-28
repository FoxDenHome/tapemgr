package mapper

import (
	"fmt"
	"log"
)

const (
	TAPE_SIZE_SPARE     = 1024 * 1024 * 1024 // 1 GB
	TAPE_SIZE_NEW_SPARE = 2 * TAPE_SIZE_SPARE

	TOMBSTONE_SIZE_SPARE = 4 * 1024 * 1024 // 4 MB
)

func (m *FileMapper) loadTapeForSize(size int64) error {
	if m.currentTape != nil && m.currentTape.Free >= size+TAPE_SIZE_SPARE {
		return nil
	}

	var err error
	if !DryRun {
		err = m.drive.Unmount()
		if err != nil {
			return fmt.Errorf("failed to unmount drive: %v", err)
		}
	}

	for _, tape := range m.inventory.GetTapes() {
		if tape.Free >= size+TAPE_SIZE_NEW_SPARE {
			log.Printf("[LOAD] Loading tape %s to drive %d", tape.Barcode, m.loaderDriveAddress)
			if DryRun {
				return nil
			}

			err = m.loader.MoveTapeToDrive(m.loaderDriveAddress, tape.Barcode)
			if err != nil {
				return fmt.Errorf("failed to move tape %s: %v", tape.Barcode, err)
			}
			err = m.drive.Mount()
			if err != nil {
				return fmt.Errorf("failed to mount tape %s in drive: %v", tape.Barcode, err)
			}
			m.currentTape = tape
			return nil
		}
	}

	// Implementation for loading tape for the given size
	return nil
}
