package inventory

import (
	"path/filepath"
	"strconv"
	"time"

	"github.com/FoxDenHome/tapemgr/scsi/drive"
	"github.com/pkg/xattr"
)

func (f *ProtoFile) IsBetterThan(other *ProtoFile) bool {
	return f.ModifiedTime.AsTime().After(other.ModifiedTime.AsTime())
}

type File interface {
	GetTape() Tape
	GetPath() string
	GetSize() int64
	GetModifiedTime() time.Time
	GetLTFSInfo(drive *drive.TapeDrive) (*FileLTFSInfo, error)
}

type file struct {
	*ProtoFile

	path string
	tape Tape
}

func (f *file) GetTape() Tape {
	return f.tape
}

func (f *file) GetPath() string {
	return f.path
}

func (f *file) GetModifiedTime() time.Time {
	return f.ModifiedTime.AsTime()
}

type FileLTFSInfo struct {
	StartBlock int
	Partition  string
}

func (f *file) GetLTFSInfo(drive *drive.TapeDrive) (*FileLTFSInfo, error) {
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

	return &FileLTFSInfo{
		StartBlock: int(startBlockNum),
		Partition:  string(partitionXattr),
	}, nil
}
