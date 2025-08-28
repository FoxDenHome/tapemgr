package mapper

import "fmt"

func (m *FileMapper) FormatTape(barcode string) error {
	err := m.formatTapeKeepMounted(barcode)
	_ = m.drive.Unmount()
	if err != nil {
		return err
	}

	return nil
}

func (m *FileMapper) formatTapeKeepMounted(barcode string) error {
	tape := m.inventory.GetOrCreateTape(barcode)

	err := m.loadTape(tape)
	if err != nil {
		return err
	}

	if DryRun {
		return nil
	}

	err = m.drive.Format(barcode)
	if err != nil {
		return fmt.Errorf("failed to format tape %s: %v", barcode, err)
	}

	err = m.drive.Mount()
	if err != nil {
		return fmt.Errorf("failed to mount tape %s: %v", barcode, err)
	}

	return m.scanCurrentTape()
}
