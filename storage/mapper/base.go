package mapper

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/FoxDenHome/tapemgr/scsi/drive"
	"github.com/FoxDenHome/tapemgr/scsi/loader"
	"github.com/FoxDenHome/tapemgr/storage/encryption"
	"github.com/FoxDenHome/tapemgr/storage/inventory"
)

type FileMapper struct {
	file *encryption.FileCryptor
	path *encryption.PathCryptor

	inventory          *inventory.Inventory
	loader             *loader.TapeLoader
	drive              *drive.TapeDrive
	loaderDriveAddress uint16

	currentTape *inventory.Tape

	sourcePrefix    string
	encryptedPrefix string

	handledFiles map[string]bool
}

func New(
	file *encryption.FileCryptor,
	path *encryption.PathCryptor,
	inventory *inventory.Inventory,
	loader *loader.TapeLoader,
	drive *drive.TapeDrive,
	sourcePrefix string,
) (*FileMapper, error) {

	encryptedPrefix := drive.MountPoint()
	if !filepath.IsAbs(sourcePrefix) {
		return nil, fmt.Errorf("source prefix %s is not absolute", sourcePrefix)
	}
	if !filepath.IsAbs(encryptedPrefix) {
		return nil, fmt.Errorf("encrypted prefix %s is not absolute", encryptedPrefix)
	}

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

		sourcePrefix:    sourcePrefix,
		encryptedPrefix: encryptedPrefix,
		handledFiles:    make(map[string]bool),
	}, nil
}

func (m *FileMapper) TombstonePath(path string) error {
	path = filepath.Clean(path)

	var err error
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path %s is not absolute", path)
	}

	relPath, err := filepath.Rel(m.sourcePrefix, path)
	if err != nil {
		return err
	}

	err = m.loadTapeForSize(TOMBSTONE_SIZE_SPARE)
	if err != nil {
		return err
	}

	encryptedRelMainPath := m.path.Encrypt(relPath) + "/"

	newFiles := make([]string, 0)

	allFiles := m.inventory.GetBestFiles()
	for encryptedPath, file := range allFiles {
		if m.handledFiles[encryptedPath] {
			continue
		}
		if !strings.HasPrefix(encryptedPath, encryptedRelMainPath) {
			continue
		}
		if file.IsTombstone() {
			continue
		}

		clearPath := m.path.Decrypt(encryptedPath)
		log.Printf("[TOMB] %s", clearPath)

		newFiles = append(newFiles, encryptedPath)

		tombPath := filepath.Join(m.encryptedPrefix, encryptedPath)
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

	return m.currentTape.AddFiles(m.drive, newFiles...)
}

func (m *FileMapper) Encrypt(path string) error {
	path = filepath.Clean(path)

	var err error
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path %s is not absolute", path)
	}

	relPath, err := filepath.Rel(m.sourcePrefix, path)
	if err != nil {
		return err
	}

	if strings.HasPrefix(relPath, "../") {
		return fmt.Errorf("path %s is outside the source prefix %s", path, m.sourcePrefix)
	}

	candidateInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	encryptedRelPath := m.path.Encrypt(relPath)
	m.handledFiles[encryptedRelPath] = true

	existingInfo := m.inventory.GetFile(encryptedRelPath)
	if existingInfo != nil && !candidateInfo.ModTime().After(existingInfo.ModifiedTime) {
		log.Printf("[SKIP] %s", path)
		return nil
	}

	encryptedPath := filepath.Join(m.encryptedPrefix, encryptedRelPath)
	log.Printf("[STOR] %s", path)

	err = m.loadTapeForSize(candidateInfo.Size())
	if err != nil {
		return err
	}

	err = m.file.EncryptMkdirAll(path, encryptedPath)
	if err != nil {
		_ = os.Remove(encryptedPath)
		_ = m.currentTape.ReloadStats(m.drive)
		return err
	}

	return m.currentTape.AddFiles(m.drive, encryptedRelPath)
}

func (m *FileMapper) Decrypt(path string) error {
	path = filepath.Clean(path)

	var err error
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path %s is not absolute", path)
	}

	relPath, err := filepath.Rel(m.encryptedPrefix, path)
	if err != nil {
		return err
	}

	if strings.HasPrefix(relPath, "../") {
		return fmt.Errorf("path %s is outside the encrypted prefix %s", path, m.sourcePrefix)
	}

	decryptedRelPath := m.path.Decrypt(relPath)
	decryptedPath := filepath.Join(m.sourcePrefix, decryptedRelPath)

	return m.file.DecryptMkdirAll(path, decryptedPath)
}
