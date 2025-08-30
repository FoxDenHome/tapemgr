package encryption

import (
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"filippo.io/age"
	"github.com/pkg/xattr"
)

const MOD_TIME_XATTR = "user.tapemgr.modtime"

type FileCryptor struct {
	identity  age.Identity
	recipient age.Recipient
}

func NewFileCryptor(identityStr string) (*FileCryptor, error) {
	identity, err := age.ParseX25519Identity(identityStr)
	if err != nil {
		return nil, err
	}
	return &FileCryptor{
		identity:  identity,
		recipient: identity.Recipient(),
	}, nil
}

func NewFileCryptorEncryptOnly(recipientStr string) (*FileCryptor, error) {
	recipient, err := age.ParseX25519Recipient(recipientStr)
	if err != nil {
		return nil, err
	}
	return &FileCryptor{
		recipient: recipient,
	}, nil
}

func (c *FileCryptor) Encrypt(src, dest string) error {
	stat, err := os.Stat(src)
	if err != nil {
		return err
	}

	err = c.encrypt(src, dest)
	if err != nil {
		_ = os.Remove(dest)
		return err
	}

	err = xattr.Set(dest, MOD_TIME_XATTR, []byte(stat.ModTime().UTC().Format(time.RFC3339)))
	if err != nil {
		log.Printf("Failed to set "+MOD_TIME_XATTR+" xattr: %v", err)
	}
	return nil
}

func (c *FileCryptor) EncryptMkdirAll(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return c.Encrypt(src, dest)
}

func copyModTimes(src, dest string) error {
	stat, err := os.Stat(src)
	if err != nil {
		_ = os.Remove(dest)
		return err
	}
	return os.Chtimes(dest, stat.ModTime(), stat.ModTime())
}

func (c *FileCryptor) Decrypt(src, dest string) error {
	err := c.decrypt(src, dest)
	if err != nil {
		_ = os.Remove(dest)
		return err
	}

	modTimeBytes, err := xattr.Get(src, MOD_TIME_XATTR)
	if err != nil {
		if !errors.Is(err, xattr.ENOATTR) {
			log.Printf("Failed to get "+MOD_TIME_XATTR+" xattr: %v", err)
		}

		return copyModTimes(src, dest)
	}

	modTime, err := time.Parse(time.RFC3339, string(modTimeBytes))
	if err != nil {
		log.Printf("Invalid "+MOD_TIME_XATTR+" xattr %s: %v", string(modTimeBytes), err)
		return copyModTimes(src, dest)
	}

	return os.Chtimes(dest, modTime, modTime)
}

func (c *FileCryptor) DecryptMkdirAll(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return c.Decrypt(src, dest)
}

func (c *FileCryptor) encrypt(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = destFile.Close() }()

	writer, err := age.Encrypt(destFile, c.recipient)
	if err != nil {
		return err
	}
	defer func() { _ = writer.Close() }()

	_, err = io.Copy(writer, srcFile)
	return err
}

func (c *FileCryptor) decrypt(src, dest string) error {
	if c.identity == nil {
		return errors.New("this FileCryptor instance is not configured for decryption")
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = destFile.Close() }()

	reader, err := age.Decrypt(srcFile, c.identity)
	if err != nil {
		return err
	}

	_, err = io.Copy(destFile, reader)
	return err
}
