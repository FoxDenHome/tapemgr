package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"strings"
)

const MAX_PART_LEN = 250

type NameCryptor struct {
	cipher cipher.Block
	iv     []byte
}

func paddedLen(len int) int {
	if len%aes.BlockSize == 0 {
		return len
	}
	len += aes.BlockSize - (len % aes.BlockSize)
	return len
}

func NewNameCryptor(key []byte) (*NameCryptor, error) {
	cipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &NameCryptor{
		cipher: cipher,
		iv:     make([]byte, aes.BlockSize),
	}, nil
}

func (c *NameCryptor) Encrypt(name string) string {
	encrypter := cipher.NewCBCEncrypter(c.cipher, c.iv)

	parts := strings.Split(name, "/")

	encryptedParts := make([]string, 0, len(parts))
	for _, part := range parts {
		encryptedPart := c.encryptPart(part, encrypter)
		for len(encryptedPart) > MAX_PART_LEN {
			encryptedParts = append(encryptedParts, encryptedPart[:MAX_PART_LEN]+",")
			encryptedPart = "," + encryptedPart[MAX_PART_LEN:]
		}
		encryptedParts = append(encryptedParts, encryptedPart)
	}
	return strings.Join(encryptedParts, "/")
}

func (c *NameCryptor) encryptPart(part string, encrypter cipher.BlockMode) string {
	paddedData := make([]byte, paddedLen(len(part)))
	copy(paddedData, part)
	encrypter.CryptBlocks(paddedData, paddedData)
	return base64.URLEncoding.EncodeToString(paddedData)
}

func (c *NameCryptor) Decrypt(name string) string {
	decrypter := cipher.NewCBCDecrypter(c.cipher, c.iv)

	nameNormalized := strings.ReplaceAll(name, ",/,", "")
	parts := strings.Split(nameNormalized, "/")

	decryptedParts := make([]string, 0, len(parts))
	for _, part := range parts {
		decryptedParts = append(decryptedParts, c.decryptPart(part, decrypter))
	}
	return strings.Join(decryptedParts, "/")
}

func (c *NameCryptor) decryptPart(part string, decrypter cipher.BlockMode) string {
	data, _ := base64.URLEncoding.DecodeString(part)
	decrypter.CryptBlocks(data, data)
	return strings.TrimRight(string(data), "\x00")
}
