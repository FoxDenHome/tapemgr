package mapper

import (
	"log"
)

func (m *FileMapper) ScanTape(barcode string) error {
	tapes := m.inventory.GetTapes()
	tape := tapes[barcode]
	if tape == nil {
		tape = m.inventory.NewTape(barcode)
	}

	err := m.loadAndMount(tape)
	if err != nil {
		return err
	}

	log.Printf("Re-inventorying tape %s", barcode)
	defer log.Printf("Finished re-inventorying tape %s", barcode)
	if DryRun {
		return nil
	}

	defer func() {
		_ = m.drive.Unmount()
	}()
	return tape.LoadFrom(m.drive)
}
