package mapper

import (
	"fmt"
	"log"
)

func (m *FileMapper) ScanTape(barcode string) error {
	tapes := m.inventory.GetTapes()
	tape := tapes[barcode]
	if tape == nil {
		return fmt.Errorf("tape %s not found", barcode)
	}

	err := m.loadAndMount(tape)
	if err != nil {
		return err
	}

	log.Printf("[SCAN] Re-inventorying tape %s", barcode)
	defer func() {
		_ = m.drive.Unmount()
	}()
	return tape.LoadFrom(m.drive)
}
