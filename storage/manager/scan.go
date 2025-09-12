package manager

import (
	"log"
)

func (m *Manager) ScanTape(barcode string) error {
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

func (m *Manager) scanCurrentTape() error {
	barcode := m.currentTape.GetBarcode()
	log.Printf("Re-inventorying tape %s", barcode)
	defer log.Printf("Finished re-inventorying tape %s", barcode)
	if DryRun {
		return nil
	}

	return m.currentTape.LoadFrom(m.drive)
}
