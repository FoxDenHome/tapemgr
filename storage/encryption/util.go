package encryption

import "os"

func copyFileTimes(src, dest string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	modTime := srcInfo.ModTime()
	return os.Chtimes(dest, modTime, modTime)
}
