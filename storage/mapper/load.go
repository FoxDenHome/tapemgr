package mapper

import "fmt"

const (
	TAPE_SIZE_SPARE     = 1024 * 1024 * 1024 // 1 GB
	TAPE_SIZE_NEW_SPARE = 2 * TAPE_SIZE_SPARE

	TOMBSTONE_SIZE_SPARE = 4 * 1024 * 1024 // 4 MB
)

func (m *FileMapper) loadTapeForSize(size int64) error {
	if m.currentTape != nil && m.currentTape.Free >= size+TAPE_SIZE_SPARE {
		return nil
	}

	for _, tape := range m.inventory.GetTapes() {
		if tape.Free >= size+TAPE_SIZE_NEW_SPARE {
			err := m.loader.MoveTapeToDrive(m.loaderDriveAddress, tape.Barcode)
			if err != nil {
				return fmt.Errorf("failed to move tape %s: %v", tape.Barcode, err)
			}
			err = m.drive.Load()
			if err != nil {
				return fmt.Errorf("failed to load tape %s into drive: %v", tape.Barcode, err)
			}
			m.currentTape = tape
			return nil
		}
	}

	// Implementation for loading tape for the given size
	return nil
}
