package manager

import (
	"fmt"
	"log"
)

func (m *Manager) MountTapeWait(barcode string) error {
	tape := m.inventory.GetOrCreateTape(barcode)

	if DryRun {
		m.currentTape = tape
		return nil
	}

	err := m.loadAndMount(tape)
	if err != nil {
		return err
	}

	log.Printf("Mounted tape %s", barcode)
	m.currentTape = tape

	err = m.scanCurrentTape()
	if err != nil {
		return err
	}

	m.drive.WaitForUnmount()
	return nil
}

func (m *Manager) UnmountAndUnload() error {
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
