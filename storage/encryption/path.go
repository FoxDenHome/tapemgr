package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/FoxDenHome/tapemgr/util"
)

type PathVersion int

const (
	PATH_VERSION_0 PathVersion = 0

	PATH_VERSION_CURRENT = PATH_VERSION_0
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
	return c.encrypt(path)
}

func (c *PathCryptor) encrypt(path string) string {
	encrypter := cipher.NewCBCEncrypter(c.cipher, c.iv)

	path = util.StripLeadingSlashes(path)

	parts := strings.Split(path, "/")

	encryptedParts := []string{}
	// TODO: Once we add a new version, do this
	// version := PATH_VERSION_CURRENT
	// encryptedParts = append(encryptedParts, fmt.Sprintf("=%d", version))
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

func (c *PathCryptor) encryptPart(part string, encrypter cipher.BlockMode) string {
	data := make([]byte, padToAESBlockSize(len(part)))
	copy(data, part)
	encrypter.CryptBlocks(data, data)
	return base64.URLEncoding.EncodeToString(data)
}

func (c *PathCryptor) Decrypt(path string) (string, error) {
	path = util.StripLeadingSlashes(path)

	version := PATH_VERSION_0
	if path[0] == '=' {
		pathSlash := strings.Index(path, "/")
		if pathSlash != -1 {
			versionInt, _ := strconv.Atoi(path[1:pathSlash])
			version = PathVersion(versionInt)
		}
		path = path[pathSlash:]
		path = util.StripLeadingSlashes(path)
	}

	switch version {
	case PATH_VERSION_0:
		return c.decrypt0(path), nil
	default:
		return "", fmt.Errorf("unknown path encryption version %d", version)
	}
}

func (c *PathCryptor) decrypt0(path string) string {
	decrypter := cipher.NewCBCDecrypter(c.cipher, c.iv)

	pathNormalized := strings.ReplaceAll(path, ",/,", "")
	parts := strings.Split(pathNormalized, "/")

	decryptedParts := make([]string, 0, len(parts))
	for _, part := range parts {
		decryptedParts = append(decryptedParts, c.decryptPart0(part, decrypter))
	}
	return strings.Join(decryptedParts, "/")
}

func (c *PathCryptor) decryptPart0(part string, decrypter cipher.BlockMode) string {
	data, _ := base64.URLEncoding.DecodeString(part)
	decrypter.CryptBlocks(data, data)
	return strings.TrimRight(string(data), "\x00")
}
