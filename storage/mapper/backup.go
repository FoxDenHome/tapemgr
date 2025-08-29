package mapper

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (m *FileMapper) Backup(target string) error {
	info, err := os.Stat(target)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return m.backupFile(target)
	}

	err = m.backupDir(target)
	if err != nil {
		return err
	}

	return m.tombstonePath(target)
}

func (m *FileMapper) backupDir(target string) error {
	entries, err := os.ReadDir(target)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		subTarget := filepath.Join(target, entry.Name())
		if entry.IsDir() {
			err = m.backupDir(subTarget)
		} else {
			err = m.backupFile(subTarget)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *FileMapper) tombstonePath(path string) error {
	path = filepath.Clean(path)

	var err error
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path %s is not absolute", path)
	}

	encryptedMainPath := m.path.Encrypt(path) + "/"

	newFiles := make([]string, 0)

	allFiles := m.inventory.GetBestFiles()
	for encryptedRelPath, file := range allFiles {
		if m.handledFiles[encryptedRelPath] {
			continue
		}
		if !strings.HasPrefix(encryptedRelPath, encryptedMainPath) {
			continue
		}
		if file.IsTombstone() {
			continue
		}

		clearRelPath := m.path.Decrypt(encryptedRelPath)
		log.Printf("[TOMB] /%s", clearRelPath)

		err = m.loadForSize(TOMBSTONE_SIZE_SPARE)
		if err != nil {
			return err
		}

		newFiles = append(newFiles, encryptedRelPath)

		if !DryRun {
			tombPath := filepath.Join(m.drive.MountPoint(), encryptedRelPath)
			tombDir := filepath.Dir(tombPath)
			err = os.MkdirAll(tombDir, 0o755)
			if err != nil {
				return err
			}
			err = os.WriteFile(tombPath, []byte{}, 0o644)
			if err != nil {
				return err
			}
		}
	}

	if DryRun || len(newFiles) == 0 {
		return nil
	}

	return m.currentTape.AddFiles(m.drive, newFiles...)
}

func (m *FileMapper) backupFile(path string) error {
	path = filepath.Clean(path)

	var err error
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path %s is not absolute", path)
	}

	candidateInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	encryptedRelPath := m.path.Encrypt(path)
	m.handledFiles[encryptedRelPath] = true

	existingInfo := m.inventory.GetFile(encryptedRelPath)
	if existingInfo != nil && (candidateInfo.ModTime().Sub(existingInfo.ModifiedTime)) < time.Second {
		// log.Printf("[SKIP] %s", path)
		return nil
	}

	encryptedPath := filepath.Join(m.drive.MountPoint(), encryptedRelPath)
	log.Printf("[STOR] %s", path)
	if existingInfo != nil {
		log.Printf("       (replacing existing backup from %s)", existingInfo.ModifiedTime.Format(time.RFC3339))
	}

	err = m.loadForSize(candidateInfo.Size())
	if err != nil {
		return err
	}

	if !DryRun {
		err = m.file.EncryptMkdirAll(path, encryptedPath)
		if err != nil {
			_ = os.Remove(encryptedPath)
			_ = m.currentTape.ReloadStats(m.drive)
			return err
		}

		return m.currentTape.AddFiles(m.drive, encryptedRelPath)
	}

	return nil
}
