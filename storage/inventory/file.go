package inventory

import (
	"path/filepath"
	"strconv"
	"time"

	"github.com/FoxDenHome/tapemgr/scsi/drive"
	"github.com/pkg/xattr"
)

type File struct {
	tape         *Tape
	path         string
	modifiedTime time.Time
	size         int64
}

type ExtendedFile struct {
	*File

	StartBlock int
	Partition  string
}

func (i *File) isTombstone() bool {
	return i.size <= 0
}

func (f *File) IsBetterThan(other *File) bool {
	return f.modifiedTime.After(other.modifiedTime)
}

func (f *File) GetTape() *Tape {
	return f.tape
}

func (f *File) GetPath() string {
	return f.path
}

func (f *File) GetSize() int64 {
	return f.size
}

func (f *File) GetModifiedTime() time.Time {
	return f.modifiedTime
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
