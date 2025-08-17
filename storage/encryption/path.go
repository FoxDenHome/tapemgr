package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"strings"
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

func (c *PathCryptor) Encrypt(path string) string {
	encrypter := cipher.NewCBCEncrypter(c.cipher, c.iv)

	parts := strings.Split(path, "/")

	encryptedParts := make([]string, 0, len(parts))
	for _, part := range parts {
		encryptedPart := c.encryptPart(part, encrypter)
		for len(encryptedPart) > c.maxPathPartLen {
			encryptedParts = append(encryptedParts, encryptedPart[:c.maxPathPartLen]+",")
			encryptedPart = "," + encryptedPart[c.maxPathPartLen:]
		}
		encryptedParts = append(encryptedParts, encryptedPart)
	}
	return strings.Join(encryptedParts, "/")
}

func (c *PathCryptor) Decrypt(path string) string {
	decrypter := cipher.NewCBCDecrypter(c.cipher, c.iv)

	pathNormalized := strings.ReplaceAll(path, ",/,", "")
	parts := strings.Split(pathNormalized, "/")

	decryptedParts := make([]string, 0, len(parts))
	for _, part := range parts {
		decryptedParts = append(decryptedParts, c.decryptPart(part, decrypter))
	}
	return strings.Join(decryptedParts, "/")
}

func (c *PathCryptor) encryptPart(part string, encrypter cipher.BlockMode) string {
	data := make([]byte, padToAESBlockSize(len(part)))
	copy(data, part)
	encrypter.CryptBlocks(data, data)
	return base64.URLEncoding.EncodeToString(data)
}

func (c *PathCryptor) decryptPart(part string, decrypter cipher.BlockMode) string {
	data, _ := base64.URLEncoding.DecodeString(part)
	decrypter.CryptBlocks(data, data)
	return strings.TrimRight(string(data), "\x00")
}
