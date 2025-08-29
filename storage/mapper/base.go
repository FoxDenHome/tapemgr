package mapper

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/FoxDenHome/tapemgr/scsi/drive"
	"github.com/FoxDenHome/tapemgr/scsi/loader"
	"github.com/FoxDenHome/tapemgr/storage/encryption"
	"github.com/FoxDenHome/tapemgr/storage/inventory"
)

var DryRun = true

type FileMapper struct {
	file *encryption.FileCryptor
	path *encryption.PathCryptor

	inventory          *inventory.Inventory
	loader             *loader.TapeLoader
	drive              *drive.TapeDrive
	loaderDriveAddress uint16

	currentTape *inventory.Tape

	handledFiles map[string]bool
}

func New(
	file *encryption.FileCryptor,
	path *encryption.PathCryptor,
	inventory *inventory.Inventory,
	loader *loader.TapeLoader,
	drive *drive.TapeDrive,
) (*FileMapper, error) {
	serialNumber, err := drive.SerialNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to get tape drive serial number: %v", err)
	}

	address, err := loader.DriveAddressBySerial(serialNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get tape drive address: %v", err)
	}

	return &FileMapper{
		file: file,
		path: path,

		inventory:          inventory,
		loader:             loader,
		drive:              drive,
		loaderDriveAddress: address,

		handledFiles: make(map[string]bool),
	}, nil
}

func (m *FileMapper) TombstonePath(path string) error {
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

	if DryRun {
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
	if existingInfo != nil && (candidateInfo.ModTime().Sub(existingInfo.ModifiedTime)).Abs() < time.Second {
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
