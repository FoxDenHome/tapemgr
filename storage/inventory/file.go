package inventory

import (
	"log"
	"path/filepath"
	"strconv"
	"time"

	"github.com/FoxDenHome/tapemgr/scsi/drive"
	"github.com/FoxDenHome/tapemgr/storage/encryption"
	"github.com/pkg/xattr"
)

type FileInfo struct {
	ModifiedTime time.Time `json:"modified_time"`
	Size         int64     `json:"size"`

	tape *Tape
	path string
}

func (i *Inventory) GetBestFiles(pathCryptor *encryption.PathCryptor) map[string]*FileInfo {
	files := make(map[string]*FileInfo)
	for _, tape := range i.tapes {
		for name, info := range tape.Files {
			clearName, err := pathCryptor.Decrypt(name)
			if err != nil {
				log.Printf("failed to decrypt path %q: %v", name, err)
				continue
			}
			oldInfo, ok := files[clearName]
			if !ok || info.IsBetterThan(oldInfo) {
				files[clearName] = info
			}
		}
	}

	for name, file := range files {
		if file.isTombstone() {
			delete(files, name)
		}
	}

	return files
}

func (i *FileInfo) isTombstone() bool {
	return i.Size <= 0
}

func (f *FileInfo) IsBetterThan(other *FileInfo) bool {
	return f.ModifiedTime.After(other.ModifiedTime)
}

func (f *FileInfo) GetTape() *Tape {
	return f.tape
}

func (f *FileInfo) GetPath() string {
	return f.path
}

type ExtendedFileInfo struct {
	Path       string
	StartBlock int
	Partition  string
}

func (f *FileInfo) GetExtended(drive *drive.TapeDrive) (*ExtendedFileInfo, error) {
	partitionXattr, err := xattr.Get(filepath.Join(drive.MountPoint(), f.path), "user.ltfs.partition")
	if err != nil {
		return nil, err
	}
	startBlockXattr, err := xattr.Get(filepath.Join(drive.MountPoint(), f.path), "user.ltfs.startblock")
	if err != nil {
		return nil, err
	}
	startBlockNum, err := strconv.ParseInt(string(startBlockXattr), 10, 64)
	if err != nil {
		return nil, err
	}

	return &ExtendedFileInfo{
		Path:       f.path,
		StartBlock: int(startBlockNum),
		Partition:  string(partitionXattr),
	}, nil
}
