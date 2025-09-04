package inventory

import (
	"encoding/json"
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

	Barcode string           `json:"barcode"`
	Files   map[string]*File `json:"files"`
	Size    int64            `json:"size"`
	Free    int64            `json:"free"`
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
		Barcode:   protoTape.Barcode,
		Size:      protoTape.Size,
		Free:      protoTape.Free,
		Files:     make(map[string]*File, len(protoTape.Files)),
	}

	for _, protoFile := range protoTape.Files {
		file := &File{
			tape: tape,
			path: protoFile.Path,

			Size:         protoFile.Size,
			ModifiedTime: protoFile.ModifiedTime.AsTime().UTC(),
		}
		file.path = util.StripLeadingSlashes(file.path)
		tape.Files[file.path] = file
	}

	return tape, nil
}

func loadFromFileJSON(inv *Inventory, filename string) (*Tape, error) {
	fh, err := os.Open(filepath.Join(inv.path, filename))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = fh.Close()
	}()

	tape := &Tape{
		inventory: inv,
	}
	dec := json.NewDecoder(fh)
	err = dec.Decode(tape)
	if err != nil {
		return nil, err
	}

	for path, file := range tape.Files {
		strippedPath := util.StripLeadingSlashes(path)
		if path != strippedPath {
			path = strippedPath
			delete(tape.Files, path)
			tape.Files[path] = file
		}
		file.tape = tape
		file.path = path
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
	t.Files = make(map[string]*File)
	err := t.reloadStats(drive)
	if err != nil {
		return err
	}

	err = t.addDir(drive, "/")
	if err != nil {
		return err
	}

	return t.Save()
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

	return t.Save()
}

func (t *Tape) addFile(drive *drive.TapeDrive, path string) error {
	path = util.StripLeadingSlashes(path)

	stat, err := os.Stat(filepath.Join(drive.MountPoint(), path))
	if err != nil {
		return err
	}

	t.Files[path] = &File{
		tape: t,
		path: path,

		Size:         stat.Size(),
		ModifiedTime: stat.ModTime().UTC(),
	}

	return nil
}

func (t *Tape) ReloadStats(drive *drive.TapeDrive) error {
	err := t.reloadStats(drive)
	if err != nil {
		return err
	}

	return t.Save()
}

func (t *Tape) reloadStats(drive *drive.TapeDrive) error {
	var stat unix.Statfs_t
	err := unix.Statfs(drive.MountPoint(), &stat)
	if err != nil {
		return err
	}

	t.Size = int64(stat.Blocks) * int64(stat.Bsize)
	t.Free = int64(stat.Bfree) * int64(stat.Bsize)

	return nil
}

func (t *Tape) Save() error {
	fh, err := os.Create(filepath.Join(t.inventory.path, t.Barcode+".proto"))
	if err != nil {
		return err
	}
	defer func() {
		_ = fh.Close()
	}()

	fileArray := make([]*ProtoFile, 0, len(t.Files))
	for _, file := range t.Files {
		fileArray = append(fileArray, &ProtoFile{
			Path:         file.path,
			Size:         file.Size,
			ModifiedTime: timestamppb.New(file.ModifiedTime),
		})
	}

	enc, err := proto.Marshal(&ProtoTape{
		Barcode: t.Barcode,
		Size:    t.Size,
		Free:    t.Free,
		Files:   fileArray,
	})
	if err != nil {
		return err
	}
	_, err = fh.Write(enc)
	return err
}
