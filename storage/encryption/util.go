package encryption

import (
	"crypto/aes"
)

func padToAESBlockSize(len int) int {
	if len%aes.BlockSize == 0 {
		return len
	}
	len += aes.BlockSize - (len % aes.BlockSize)
	return len
}
