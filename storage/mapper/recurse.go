package mapper

import (
	"os"
	"path/filepath"
)

func (m *FileMapper) EncryptRecursive(target string) error {
	info, err := os.Stat(target)
	if err != nil {
		return err
	}

	if info.IsDir() {
		entries, err := os.ReadDir(target)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			err = m.EncryptRecursive(filepath.Join(target, entry.Name()))
			if err != nil {
				return err
			}
		}
	} else {
		err = m.Encrypt(target)
		if err != nil {
			return err
		}
	}

	return nil
}
