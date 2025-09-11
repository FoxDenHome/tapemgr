package inventory

import (
	"log"
	"os"
	"path/filepath"

	"github.com/FoxDenHome/tapemgr/scsi/drive"
	"github.com/FoxDenHome/tapemgr/util"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/proto"
)

type Tape struct {
	inventory *Inventory

	Barcode string `json:"barcode"`
	Size    int64  `json:"size"`
	Free    int64  `json:"free"`
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
	}

	_, err = inv.db.Exec("INSERT IGNORE INTO tapes (barcode, size, free) VALUES (?, ?, ?)", tape.Barcode, tape.Size, tape.Free)
	if err != nil {
		return nil, err
	}

	for _, protoFile := range protoTape.Files {
		file := &File{
			barcode:      tape.Barcode,
			path:         protoFile.Path,
			Size:         protoFile.Size,
			ModifiedTime: protoFile.ModifiedTime.AsTime().UTC(),
		}
		file.path = util.StripLeadingSlashes(file.path)
		_, err = inv.db.Exec("INSERT IGNORE INTO files (path, barcode, size, modified_time) VALUES (?, ?, ?, ?)", file.path, tape.Barcode, file.Size, file.ModifiedTime)
		if err != nil {
			return nil, err
		}
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
	err := t.reloadStats(drive)
	if err != nil {
		return err
	}

	err = t.addDir(drive, "/")
	if err != nil {
		return err
	}

	return nil
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

	return nil
}

func (t *Tape) addFile(drive *drive.TapeDrive, path string) error {
	path = util.StripLeadingSlashes(path)

	stat, err := os.Stat(filepath.Join(drive.MountPoint(), path))
	if err != nil {
		return err
	}

	_, err = t.inventory.db.Exec("INSERT OR REPLACE INTO files (path, barcode, size, modified_time) VALUES (?, ?, ?, ?)", path, t.Barcode, stat.Size(), stat.ModTime().UTC())
	return err
}

func (t *Tape) ReloadStats(drive *drive.TapeDrive) error {
	err := t.reloadStats(drive)
	if err != nil {
		return err
	}

	return nil
}

func (t *Tape) reloadStats(drive *drive.TapeDrive) error {
	var stat unix.Statfs_t
	err := unix.Statfs(drive.MountPoint(), &stat)
	if err != nil {
		return err
	}

	t.Size = int64(stat.Blocks) * int64(stat.Bsize)
	t.Free = int64(stat.Bfree) * int64(stat.Bsize)

	_, err = t.inventory.db.Exec("INSERT OR REPLACE INTO tapes (barcode, size, free) VALUES (?, ?, ?)", t.Barcode, t.Size, t.Free)
	return err
}
