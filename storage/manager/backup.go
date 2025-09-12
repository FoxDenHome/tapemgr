package manager

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/FoxDenHome/tapemgr/storage/inventory"
	"github.com/FoxDenHome/tapemgr/util"
)

func (m *Manager) Backup(targets ...string) error {
	bestFiles := m.inventory.GetBestFiles(m.path)

	for _, target := range targets {
		log.Printf("Backing up target %v", target)

		info, err := os.Stat(target)
		if err != nil {
			return err
		}

		handledFiles := make(map[string]bool)

		if !info.IsDir() {
			return m.backupFile(target, handledFiles, bestFiles)
		}

		err = m.backupDir(target, handledFiles, bestFiles)
		if err != nil {
			return err
		}

		err = m.tombstonePath(target, handledFiles, bestFiles)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) backupDir(target string, handledFiles map[string]bool, bestFiles map[string]*inventory.File) error {
	entries, err := os.ReadDir(target)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		subTarget := filepath.Join(target, entry.Name())
		if entry.IsDir() {
			err = m.backupDir(subTarget, handledFiles, bestFiles)
		} else {
			err = m.backupFile(subTarget, handledFiles, bestFiles)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) tombstonePath(path string, handledFiles map[string]bool, bestFiles map[string]*inventory.File) error {
	path = filepath.Clean(path)

	var err error
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path %s is not absolute", path)
	}

	mainPath := m.path.Encrypt(path) + "/"

	newFiles := make([]string, 0)
	for clearRelPath := range bestFiles {
		if handledFiles[clearRelPath] {
			continue
		}
		if !strings.HasPrefix(clearRelPath, mainPath) {
			continue
		}

		log.Printf("[TOMB] /%s", clearRelPath)
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

func (m *Manager) backupFile(path string, handledFiles map[string]bool, bestFiles map[string]*inventory.File) error {
	path = filepath.Clean(path)

	if !filepath.IsAbs(path) {
		return fmt.Errorf("path %s is not absolute", path)
	}

	candidateInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	encryptedRelPath := m.path.Encrypt(path)

	relPath := util.StripLeadingSlashes(path)
	existingInfo := bestFiles[relPath]

	if handledFiles[relPath] {
		// Same file twice in a backup job
		return nil
	}

	handledFiles[relPath] = true

	if existingInfo != nil && (candidateInfo.ModTime().Sub(existingInfo.GetModifiedTime())) < time.Second {
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
