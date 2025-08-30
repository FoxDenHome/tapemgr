package inventory

import (
	"path/filepath"
	"strconv"
	"time"

	"github.com/FoxDenHome/tapemgr/scsi/drive"
	"github.com/pkg/xattr"
)

type File struct {
	ModifiedTime time.Time `json:"modified_time"`
	Size         int64     `json:"size"`

	tape *Tape
	path string
}

type ExtendedFile struct {
	*File

	StartBlock int
	Partition  string
}

func (i *File) isTombstone() bool {
	return i.Size <= 0
}

func (f *File) IsBetterThan(other *File) bool {
	return f.ModifiedTime.After(other.ModifiedTime)
}

func (f *File) GetTape() *Tape {
	return f.tape
}

func (f *File) GetPath() string {
	return f.path
}

func (f *File) GetExtended(drive *drive.TapeDrive) (*ExtendedFile, error) {
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

	return &ExtendedFile{
		File:       f,
		StartBlock: int(startBlockNum),
		Partition:  string(partitionXattr),
	}, nil
}
