package mapper

import (
	"log"
)

func (m *FileMapper) ScanTape(barcode string) error {
	tape := m.inventory.GetOrCreateTape(barcode)

	err := m.loadAndMount(tape)
	if err != nil {
		return err
	}

	defer func() {
		_ = m.drive.Unmount()
	}()
	return m.scanCurrentTape()
}

func (m *FileMapper) scanCurrentTape() error {
	log.Printf("Re-inventorying tape %s", m.currentTape.Barcode)
	defer log.Printf("Finished re-inventorying tape %s", m.currentTape.Barcode)
	if DryRun {
		return nil
	}

	return m.currentTape.LoadFrom(m.drive)
}
