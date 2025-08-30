package manager

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (m *Manager) Backup(target string) error {
	info, err := os.Stat(target)
	if err != nil {
		return err
	}

	handledFiles := make(map[string]bool)

	if !info.IsDir() {
		return m.backupFile(target, handledFiles)
	}

	err = m.backupDir(target, handledFiles)
	if err != nil {
		return err
	}

	return m.tombstonePath(target, handledFiles)
}

func (m *Manager) backupDir(target string, handledFiles map[string]bool) error {
	entries, err := os.ReadDir(target)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		subTarget := filepath.Join(target, entry.Name())
		if entry.IsDir() {
			err = m.backupDir(subTarget, handledFiles)
		} else {
			err = m.backupFile(subTarget, handledFiles)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) tombstonePath(path string, handledFiles map[string]bool) error {
	path = filepath.Clean(path)

	var err error
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path %s is not absolute", path)
	}

	mainPath := m.path.Encrypt(path) + "/"

	newFiles := make([]string, 0)

	allFiles := m.inventory.GetBestFiles(m.path)
	for clearRelPath := range allFiles {
		clearAbsPath := filepath.Join("/", clearRelPath)
		if handledFiles[clearAbsPath] {
			continue
		}
		if !strings.HasPrefix(clearRelPath, mainPath) {
			continue
		}

		log.Printf("[TOMB] %s", clearAbsPath)
		encryptedRelPath := m.path.Encrypt(clearRelPath)

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

func (m *Manager) backupFile(path string, handledFiles map[string]bool) error {
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
	handledFiles[path] = true

	existingInfo := m.inventory.GetFile(path, m.path)
	if existingInfo != nil && (candidateInfo.ModTime().Sub(existingInfo.ModifiedTime)) < time.Second {
		// log.Printf("[SKIP] %s", path)
		return nil
	}

	encryptedPath := filepath.Join(m.drive.MountPoint(), encryptedRelPath)
	log.Printf("[STOR] %s", path)

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
