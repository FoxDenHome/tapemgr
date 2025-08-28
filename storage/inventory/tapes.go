package inventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"

	"github.com/FoxDenHome/tapemgr/scsi/drive"
	"golang.org/x/sys/unix"

	"github.com/pkg/xattr"
)

func LoadFromFile(filename string) (*Tape, error) {
	fh, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	tape := &Tape{}
	dec := json.NewDecoder(fh)
	err = dec.Decode(tape)
	if err != nil {
		return nil, err
	}
	for name, file := range tape.Files {
		if name[0] == '/' {
			name = name[1:]
			delete(tape.Files, name)
			tape.Files[name] = file
		}
		file.tape = tape
		file.name = name
	}
	return tape, nil
}

func (t *Tape) addDir(drive *drive.TapeDrive, path string) error {
	entries, err := os.ReadDir(filepath.Join(drive.MountPoint(), path))
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			err = t.addDir(drive, entryPath)
		} else {
			err = t.AddFiles(drive, entryPath)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Tape) LoadFrom(drive *drive.TapeDrive) error {
	t.Files = make(map[string]*FileInfo)
	err := t.ReloadStats(drive)
	if err != nil {
		return err
	}
	return t.addDir(drive, "/")
}

func (t *Tape) AddFiles(drive *drive.TapeDrive, path ...string) error {
	err := t.ReloadStats(drive)
	if err != nil {
		return err
	}

	for _, p := range path {
		err = t.addFile(drive, p)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Tape) addFile(drive *drive.TapeDrive, path string) error {
	mountPoint := drive.MountPoint()

	stat, err := os.Stat(filepath.Join(mountPoint, path))
	if err != nil {
		return err
	}

	partitionXattr, err := xattr.Get(filepath.Join(mountPoint, path), "user.ltfs.partition")
	if err != nil {
		return err
	}
	startBlockXattr, err := xattr.Get(filepath.Join(mountPoint, path), "user.ltfs.startblock")
	if err != nil {
		return err
	}
	startBlockNum, err := strconv.ParseInt(string(startBlockXattr), 10, 64)
	if err != nil {
		return err
	}

	info := &FileInfo{
		tape: t,
		name: path,

		Size:         stat.Size(),
		ModifiedTime: stat.ModTime(),

		Partition:  string(partitionXattr),
		StartBlock: startBlockNum,
	}
	t.Files[path] = info

	return nil
}

func (t *Tape) ReloadStats(drive *drive.TapeDrive) error {
	var stat unix.Statfs_t
	err := unix.Statfs(drive.MountPoint(), &stat)
	if err != nil {
		return err
	}

	t.Size = int64(stat.Blocks) * int64(stat.Bsize)
	t.Free = int64(stat.Bfree) * int64(stat.Bsize)

	return nil
}

func (i *Inventory) GetTapes() map[string]*Tape {
	return i.tapes
}
