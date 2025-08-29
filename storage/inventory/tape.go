package inventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/FoxDenHome/tapemgr/scsi/drive"
	"golang.org/x/sys/unix"
)

func LoadFromFile(inv *Inventory, filename string) (*Tape, error) {
	fh, err := os.Open(filepath.Join(inv.path, filename))
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	tape := &Tape{
		inventory: inv,
	}
	dec := json.NewDecoder(fh)
	err = dec.Decode(tape)
	if err != nil {
		return nil, err
	}
	for path, file := range tape.Files {
		if path[0] == '/' {
			path = strings.TrimPrefix(path, "/")
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
	t.Files = make(map[string]*FileInfo)
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
	mountPoint := drive.MountPoint()

	stat, err := os.Stat(filepath.Join(mountPoint, path))
	if err != nil {
		return err
	}

	if path[0] == '/' {
		path = strings.TrimPrefix(path, "/")
	}

	t.Files[path] = &FileInfo{
		tape: t,
		path: path,

		Size:         stat.Size(),
		ModifiedTime: stat.ModTime(),
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
	fh, err := os.Create(filepath.Join(t.inventory.path, t.Barcode+".json"))
	if err != nil {
		return err
	}
	defer fh.Close()

	enc := json.NewEncoder(fh)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "    ")
	return enc.Encode(t)
}
