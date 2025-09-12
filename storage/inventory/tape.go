package inventory

import (
	"log"
	"os"
	"path/filepath"

	"github.com/FoxDenHome/tapemgr/scsi/drive"
	"github.com/FoxDenHome/tapemgr/util"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Tape struct {
	inventory *Inventory

	barcode string
	files   map[string]*File
	size    int64
	free    int64
}

func (t *Tape) GetBarcode() string {
	return t.barcode
}

func (t *Tape) GetSize() int64 {
	return t.size
}

func (t *Tape) GetFree() int64 {
	return t.free
}

func loadFromFileProto(inv *Inventory, filename string) (*Tape, error) {
	log.Printf("Loading tape inventory from %s", filename)
	data, err := os.ReadFile(filepath.Join(inv.path, filename))
	if err != nil {
		return nil, err
	}

	protoTape := ProtoTape{}
	err = proto.Unmarshal(data, &protoTape)
	if err != nil {
		return nil, err
	}

	tape := &Tape{
		inventory: inv,
		barcode:   protoTape.Barcode,
		size:      protoTape.Size,
		free:      protoTape.Free,
		files:     make(map[string]*File, len(protoTape.Files)),
	}

	for _, protoFile := range protoTape.Files {
		file := &File{
			tape:         tape,
			path:         protoFile.Path,
			size:         protoFile.Size,
			modifiedTime: protoFile.ModifiedTime.AsTime().UTC(),
		}
		file.path = util.StripLeadingSlashes(file.path)
		tape.files[file.path] = file
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
			err = t.addFile(drive, entryPath)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Tape) LoadFrom(drive *drive.TapeDrive) error {
	t.files = make(map[string]*File)
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

func (t *Tape) AddFiles(drive *drive.TapeDrive, path ...string) error {
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

func (t *Tape) addFile(drive *drive.TapeDrive, path string) error {
	path = util.StripLeadingSlashes(path)

	stat, err := os.Stat(filepath.Join(drive.MountPoint(), path))
	if err != nil {
		return err
	}

	t.files[path] = &File{
		tape: t,
		path: path,

		size:         stat.Size(),
		modifiedTime: stat.ModTime().UTC(),
	}

	return nil
}

func (t *Tape) ReloadStats(drive *drive.TapeDrive) error {
	err := t.reloadStats(drive)
	if err != nil {
		return err
	}

	return t.save()
}

func (t *Tape) reloadStats(drive *drive.TapeDrive) error {
	var stat unix.Statfs_t
	err := unix.Statfs(drive.MountPoint(), &stat)
	if err != nil {
		return err
	}

	t.size = int64(stat.Blocks) * int64(stat.Bsize)
	t.free = int64(stat.Bfree) * int64(stat.Bsize)

	return nil
}

func (t *Tape) save() error {
	fh, err := os.Create(filepath.Join(t.inventory.path, t.barcode+".proto"))
	if err != nil {
		return err
	}
	defer func() {
		_ = fh.Close()
	}()

	fileArray := make([]*ProtoFile, 0, len(t.files))
	for _, file := range t.files {
		fileArray = append(fileArray, &ProtoFile{
			Path:         file.path,
			Size:         file.size,
			ModifiedTime: timestamppb.New(file.modifiedTime),
		})
	}

	enc, err := proto.Marshal(&ProtoTape{
		Barcode: t.barcode,
		Size:    t.size,
		Free:    t.free,
		Files:   fileArray,
	})
	if err != nil {
		return err
	}
	_, err = fh.Write(enc)
	return err
}
