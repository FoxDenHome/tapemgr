package inventory

import (
	"os"
	"path/filepath"

	"github.com/FoxDenHome/tapemgr/scsi/drive"
	"github.com/FoxDenHome/tapemgr/util"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Tape interface {
	GetBarcode() string
	GetSize() int64
	GetFree() int64
	GetFiles() map[string]*ProtoFile
	LoadFrom(drive *drive.TapeDrive) error
	AddFiles(drive *drive.TapeDrive, path ...string) error
	ReloadStats(drive *drive.TapeDrive) error
	Equals(other Tape) bool
}

type tape struct {
	ProtoTape

	inventory *Inventory
}

func loadFromFileProto(inv *Inventory, filename string) (*tape, error) {
	data, err := os.ReadFile(filepath.Join(inv.path, filename))
	if err != nil {
		return nil, err
	}

	tp := &tape{
		inventory: inv,
	}

	err = proto.Unmarshal(data, &tp.ProtoTape)
	if err != nil {
		return nil, err
	}

	return tp, nil
}

func (t *tape) addDir(drive *drive.TapeDrive, path string) error {
	entries, err := os.ReadDir(filepath.Join(drive.MountPoint(), path))
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			err = t.addDir(drive, entryPath)
		} else {
			err = t.addFile(drive, entryPath)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *tape) LoadFrom(drive *drive.TapeDrive) error {
	t.Files = make(map[string]*ProtoFile)
	err := t.reloadStats(drive)
	if err != nil {
		return err
	}

	err = t.addDir(drive, "/")
	if err != nil {
		return err
	}

	return t.save()
}

func (t *tape) AddFiles(drive *drive.TapeDrive, path ...string) error {
	err := t.reloadStats(drive)
	if err != nil {
		return err
	}

	for _, p := range path {
		err = t.addFile(drive, p)
		if err != nil {
			return err
		}
	}

	return t.save()
}

func (t *tape) addFile(drive *drive.TapeDrive, path string) error {
	path = util.StripLeadingSlashes(path)

	stat, err := os.Stat(filepath.Join(drive.MountPoint(), path))
	if err != nil {
		return err
	}

	t.Files[path] = &ProtoFile{
		Size:         stat.Size(),
		ModifiedTime: timestamppb.New(stat.ModTime().UTC()),
	}

	return nil
}

func (t *tape) ReloadStats(drive *drive.TapeDrive) error {
	err := t.reloadStats(drive)
	if err != nil {
		return err
	}

	return t.save()
}

func (t *tape) reloadStats(drive *drive.TapeDrive) error {
	var stat unix.Statfs_t
	err := unix.Statfs(drive.MountPoint(), &stat)
	if err != nil {
		return err
	}

	t.Size = int64(stat.Blocks) * int64(stat.Bsize)
	t.Free = int64(stat.Bfree) * int64(stat.Bsize)

	return nil
}

func (t *tape) save() error {
	fh, err := os.Create(filepath.Join(t.inventory.path, t.Barcode+".proto"))
	if err != nil {
		return err
	}
	defer func() {
		_ = fh.Close()
	}()

	enc, err := proto.Marshal(&t.ProtoTape)
	if err != nil {
		return err
	}
	_, err = fh.Write(enc)
	return err
}

func (t *tape) Equals(other Tape) bool {
	if other == nil {
		return false
	}
	return t.Barcode == other.GetBarcode()
}
