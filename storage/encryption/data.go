package encryption

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"filippo.io/age"
)

type FileCryptor struct {
	identity  age.Identity
	recipient age.Recipient
}

func copyFileTimes(src, dest string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	modTime := srcInfo.ModTime()
	return os.Chtimes(dest, modTime, modTime)
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

func (c *FileCryptor) Encrypt(dst io.Writer) (io.WriteCloser, error) {
	return age.Encrypt(dst, c.recipient)
}

func (c *FileCryptor) EncryptFile(src, dest string) error {
	err := c.encryptFile(src, dest)
	if err != nil {
		_ = os.Remove(dest)
		return err
	}
	return copyFileTimes(src, dest)
}

func (c *FileCryptor) encryptFile(src, dest string) error {
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

	writer, err := c.Encrypt(destFile)
	if err != nil {
		return err
	}
	defer func() { _ = writer.Close() }()

	_, err = io.Copy(writer, srcFile)
	return err
}

func (c *FileCryptor) EncryptFileMakedirs(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return c.EncryptFile(src, dest)
}

func (c *FileCryptor) Decrypt(src io.Reader) (io.Reader, error) {
	if c.identity == nil {
		return nil, errors.New("this FileCryptor instance is not configured for decryption")
	}
	return age.Decrypt(src, c.identity)
}

func (c *FileCryptor) DecryptFile(src, dest string) error {
	err := c.decryptFile(src, dest)
	if err != nil {
		_ = os.Remove(dest)
		return err
	}
	return copyFileTimes(src, dest)
}

func (c *FileCryptor) decryptFile(src, dest string) error {
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

	reader, err := c.Decrypt(srcFile)
	if err != nil {
		return err
	}

	_, err = io.Copy(destFile, reader)
	return err
}

func (c *FileCryptor) DecryptFileMakedirs(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return c.DecryptFile(src, dest)
}
