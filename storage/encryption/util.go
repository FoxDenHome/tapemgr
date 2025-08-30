package encryption

import (
	"crypto/aes"
)

func padToAESBlockSize(len int, alwaysPad bool) int {
	if len%aes.BlockSize == 0 && !alwaysPad {
		return len
	}
	len += aes.BlockSize - (len % aes.BlockSize)
	return len
}
