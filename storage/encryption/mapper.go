package encryption

import (
	"fmt"
	"path/filepath"
	"strings"
)

type MappedCryptor struct {
	file            *FileCryptor
	path            *PathCryptor
	sourcePrefix    string
	encryptedPrefix string
}

func NewMappedCryptor(file *FileCryptor, path *PathCryptor, sourcePrefix, encryptedPrefix string) (*MappedCryptor, error) {
	return &MappedCryptor{
		file:            file,
		path:            path,
		sourcePrefix:    sourcePrefix,
		encryptedPrefix: encryptedPrefix,
	}, nil
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

	encryptedRelPath := c.path.Encrypt(relPath)
	encryptedPath := filepath.Join(c.encryptedPrefix, encryptedRelPath)

	return c.file.Encrypt(path, encryptedPath)
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

	return c.file.Decrypt(path, decryptedPath)
}
