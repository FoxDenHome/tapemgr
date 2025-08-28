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

	tapes := m.inventory.GetTapes()

	for _, tape := range tapes {
		if tape.Free >= size+TAPE_SIZE_NEW_SPARE {
			return m.loadAndMount(tape)
		}
	}

	volumeTags, err := m.loader.GetVolumeTags()
	if err != nil {
		return fmt.Errorf("failed to get volume tags: %v", err)
	}

	for _, barcode := range volumeTags {
		if tapes[barcode] == nil {
			// Found unused new tape!
			return m.formatTapeKeepMounted(barcode)
		}
	}

	return nil
}

func (m *FileMapper) loadTape(tape *inventory.Tape) error {
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

	m.currentTape = tape
	return nil
}

func (m *FileMapper) loadAndMount(tape *inventory.Tape) error {
	if m.currentTape != nil && m.currentTape.Barcode == tape.Barcode {
		return nil
	}

	err := m.loadTape(tape)
	if err != nil {
		return err
	}

	if DryRun {
		return nil
	}

	err = m.drive.Mount()
	if err != nil {
		return fmt.Errorf("failed to mount tape %s in drive: %v", tape.Barcode, err)
	}

	return nil
}
