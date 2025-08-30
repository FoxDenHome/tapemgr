package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"strings"

	"github.com/FoxDenHome/tapemgr/util"
)

type Version int

const (
	VERSION_1 Version = 1
	VERSION_2 Version = 2

	VERSION_MIN    = VERSION_1
	VERSION_LATEST = VERSION_2
)

type PathCryptor struct {
	maxPathPartLen int
	cipher         cipher.Block
	iv             []byte
}

func NewPathCryptor(key []byte) (*PathCryptor, error) {
	cipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &PathCryptor{
		cipher:         cipher,
		maxPathPartLen: 250,
		iv:             make([]byte, aes.BlockSize),
	}, nil
}

func (c *PathCryptor) EncryptVersion(path string, version Version) string {
	return c.encrypt(path, version)
}

func (c *PathCryptor) Encrypt(path string) string {
	return c.encrypt(path, VERSION_LATEST)
}

func (c *PathCryptor) encrypt(path string, version Version) string {
	encrypter := cipher.NewCBCEncrypter(c.cipher, c.iv)

	path, _ = util.StripLeadingSlashes(path)

	parts := strings.Split(path, "/")

	encryptedParts := make([]string, 0, len(parts))
	for _, part := range parts {
		encryptedPart := c.encryptPart(part, version, encrypter)
		for len(encryptedPart) > c.maxPathPartLen {
			encryptedParts = append(encryptedParts, encryptedPart[:c.maxPathPartLen]+",")
			encryptedPart = "," + encryptedPart[c.maxPathPartLen:]
		}
		encryptedParts = append(encryptedParts, encryptedPart)
	}
	return strings.Join(encryptedParts, "/")
}

func (c *PathCryptor) Decrypt(path string) string {
	// If we add more versions, got to auto-determine!
	version := VERSION_LATEST

	decrypter := cipher.NewCBCDecrypter(c.cipher, c.iv)

	path, _ = util.StripLeadingSlashes(path)

	pathNormalized := strings.ReplaceAll(path, ",/,", "")
	parts := strings.Split(pathNormalized, "/")

	decryptedParts := make([]string, 0, len(parts))
	for _, part := range parts {
		decryptedParts = append(decryptedParts, c.decryptPart(part, version, decrypter))
	}
	return strings.Join(decryptedParts, "/")
}

func (c *PathCryptor) encryptPart(part string, version Version, encrypter cipher.BlockMode) string {
	data := make([]byte, padToAESBlockSize(len(part), version == VERSION_1))
	copy(data, part)
	encrypter.CryptBlocks(data, data)
	return base64.URLEncoding.EncodeToString(data)
}

func (c *PathCryptor) decryptPart(part string, _ Version, decrypter cipher.BlockMode) string {
	data, _ := base64.URLEncoding.DecodeString(part)
	decrypter.CryptBlocks(data, data)
	return strings.TrimRight(string(data), "\x00")
}
