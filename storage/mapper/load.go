package mapper

import (
	"fmt"
	"log"

	"github.com/FoxDenHome/tapemgr/storage/inventory"
)

const (
	TAPE_SIZE_SPARE     = 1024 * 1024 * 1024 // 1 GB
	TAPE_SIZE_NEW_SPARE = 2 * TAPE_SIZE_SPARE

	TOMBSTONE_SIZE_SPARE = 4 * 1024 * 1024 // 4 MB
)

func (m *FileMapper) loadForSize(size int64) error {
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
			return m.loadAndMount(tape)
		}
	}

	return nil
}

func (m *FileMapper) loadAndMount(tape *inventory.Tape) error {
	if m.currentTape != nil && m.currentTape.Barcode == tape.Barcode {
		return nil
	}

	log.Printf("Loading tape %s to drive %d", tape.Barcode, m.loaderDriveAddress)

	if DryRun {
		m.currentTape = tape
		return nil
	}

	err := m.drive.Unmount()
	if err != nil {
		return fmt.Errorf("failed to unmount drive: %v", err)
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

func (m *FileMapper) UnmountAndUnload() error {
	m.currentTape = nil
	if DryRun {
		return nil
	}

	err := m.drive.Unmount()
	if err != nil {
		return fmt.Errorf("unmounting drive: %w", err)
	}

	err = m.loader.MoveDriveTapeToStorage(m.loaderDriveAddress)
	if err != nil {
		return fmt.Errorf("moving tape from drive %d to storage: %w", m.loaderDriveAddress, err)
	}

	return nil
}
