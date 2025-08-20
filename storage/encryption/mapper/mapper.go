package mapper

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/FoxDenHome/tapemgr/storage/encryption"
	"github.com/FoxDenHome/tapemgr/storage/inventory"
)

type MappedCryptor struct {
	file            *encryption.FileCryptor
	path            *encryption.PathCryptor
	inventory       *inventory.Inventory
	sourcePrefix    string
	encryptedPrefix string

	handledFiles map[string]bool
}

func New(file *encryption.FileCryptor, path *encryption.PathCryptor, inventory *inventory.Inventory, sourcePrefix, encryptedPrefix string) (*MappedCryptor, error) {
	if !filepath.IsAbs(sourcePrefix) {
		return nil, fmt.Errorf("source prefix %s is not absolute", sourcePrefix)
	}
	if !filepath.IsAbs(encryptedPrefix) {
		return nil, fmt.Errorf("encrypted prefix %s is not absolute", encryptedPrefix)
	}

	return &MappedCryptor{
		file:            file,
		path:            path,
		inventory:       inventory,
		sourcePrefix:    sourcePrefix,
		encryptedPrefix: encryptedPrefix,
		handledFiles:    make(map[string]bool),
	}, nil
}

func (c *MappedCryptor) TombstonePath(path string) error {
	path = filepath.Clean(path)

	var err error
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path %s is not absolute", path)
	}

	relPath, err := filepath.Rel(c.sourcePrefix, path)
	if err != nil {
		return err
	}

	encryptedRelMainPath := c.path.Encrypt(relPath) + "/"

	allFiles := c.inventory.GetBestFiles()
	for encryptedPath, file := range allFiles {
		if c.handledFiles[encryptedPath] {
			continue
		}
		if !strings.HasPrefix(encryptedPath, encryptedRelMainPath) {
			continue
		}
		if file.IsTombstone() {
			continue
		}

		clearPath := c.path.Decrypt(encryptedPath)
		log.Printf("[TOMB] %s", clearPath)

		tombPath := filepath.Join(c.encryptedPrefix, encryptedPath)
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

	return nil
}

func (c *MappedCryptor) Encrypt(path string) error {
	path = filepath.Clean(path)

	var err error
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path %s is not absolute", path)
	}

	relPath, err := filepath.Rel(c.sourcePrefix, path)
	if err != nil {
		return err
	}

	if strings.HasPrefix(relPath, "../") {
		return fmt.Errorf("path %s is outside the source prefix %s", path, c.sourcePrefix)
	}

	candidateInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	encryptedRelPath := c.path.Encrypt(relPath)
	c.handledFiles[encryptedRelPath] = true

	existingInfo := c.inventory.GetFile(encryptedRelPath)
	if existingInfo != nil && !candidateInfo.ModTime().After(existingInfo.ModifiedTime) {
		log.Printf("[SKIP] %s", path)
		return nil
	}

	encryptedPath := filepath.Join(c.encryptedPrefix, encryptedRelPath)
	log.Printf("[STOR] %s", path)
	return c.file.EncryptMkdirAll(path, encryptedPath)
}

func (c *MappedCryptor) Decrypt(path string) error {
	path = filepath.Clean(path)

	var err error
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path %s is not absolute", path)
	}

	relPath, err := filepath.Rel(c.encryptedPrefix, path)
	if err != nil {
		return err
	}

	if strings.HasPrefix(relPath, "../") {
		return fmt.Errorf("path %s is outside the encrypted prefix %s", path, c.sourcePrefix)
	}

	decryptedRelPath := c.path.Decrypt(relPath)
	decryptedPath := filepath.Join(c.sourcePrefix, decryptedRelPath)

	return c.file.DecryptMkdirAll(path, decryptedPath)
}
