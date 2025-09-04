package encryption

import (
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/pkg/xattr"
)

const (
	XATTR_MOD_TIME = "user.tapemgr.modtime"
	XATTR_MODE     = "user.tapemgr.mode"
)

func copyModTimes(src, dest string) error {
	stat, err := os.Stat(src)
	if err != nil {
		_ = os.Remove(dest)
		return err
	}

	return os.Chtimes(dest, stat.ModTime(), stat.ModTime())
}

func copyModTimesXattr(src, dest string) error {
	modTimeBytes, err := xattr.Get(src, XATTR_MOD_TIME)
	if err != nil {
		if !errors.Is(err, xattr.ENOATTR) {
			log.Printf("Failed to get "+XATTR_MOD_TIME+" xattr: %v", err)
		}
		return copyModTimes(src, dest)
	}

	modTime, err := time.Parse(time.RFC3339, string(modTimeBytes))
	if err != nil {
		log.Printf("Invalid "+XATTR_MOD_TIME+" xattr %s: %v", string(modTimeBytes), err)
		return copyModTimes(src, dest)
	}

	return os.Chtimes(dest, modTime, modTime)
}

func copyModeXattr(src, dest string) error {
	modeBytes, err := xattr.Get(src, XATTR_MODE)
	if err != nil {
		if !errors.Is(err, xattr.ENOATTR) {
			log.Printf("Failed to get "+XATTR_MODE+" xattr: %v", err)
		}
		return nil
	}

	mode, err := strconv.ParseInt(string(modeBytes), 8, 32)
	if err != nil {
		log.Printf("Invalid "+XATTR_MODE+" xattr %s: %v", string(modeBytes), err)
		return nil
	}

	return os.Chmod(dest, os.FileMode(mode))
}

// TODO: Implement more generic xattr loaders and storers

func generateXattr(src, dest string) error {
	stat, err := os.Stat(src)
	if err != nil {
		return err
	}

	err = xattr.Set(dest, XATTR_MOD_TIME, []byte(stat.ModTime().UTC().Format(time.RFC3339)))
	if err != nil {
		log.Printf("Failed to set "+XATTR_MOD_TIME+" xattr: %v", err)
	}
	err = xattr.Set(dest, XATTR_MODE, []byte(strconv.Itoa(int(stat.Mode()))))
	if err != nil {
		log.Printf("Failed to set "+XATTR_MODE+" xattr: %v", err)
	}

	return nil
}

func retrieveXattr(src, dest string) error {
	err := copyModTimesXattr(src, dest)
	if err != nil {
		return err
	}

	err = copyModeXattr(src, dest)
	if err != nil {
		return err
	}

	return nil
}
