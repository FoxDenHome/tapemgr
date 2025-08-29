package encryption

import (
	"crypto/aes"
	"os"
)

func copyFileModTime(src, dest string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	modTime := srcInfo.ModTime()
	return os.Chtimes(dest, modTime, modTime)
}

func padToAESBlockSize(len int) int {
	// if len%aes.BlockSize == 0 {
	// 	return len
	// }
	len += aes.BlockSize - (len % aes.BlockSize)
	return len
}
