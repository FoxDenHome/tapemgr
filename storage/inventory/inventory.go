package inventory

import (
	"log"
	"os"
	"slices"
	"strings"

	"github.com/FoxDenHome/tapemgr/storage/encryption"
)

//go:generate go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
//go:generate protoc --go_out=. --go_opt=paths=source_relative inventory.proto

type Inventory struct {
	path  string
	tapes map[string]*Tape
}

func New(path string) (*Inventory, error) {
	inv := &Inventory{
		path:  path,
		tapes: make(map[string]*Tape),
	}
	return inv, inv.Reload()
}

func (i *Inventory) loadTapeList(suffix string, files []os.DirEntry, loader func(i *Inventory, filename string) (*Tape, error)) {
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if !strings.HasSuffix(name, suffix) {
			continue
		}

		barcode := strings.TrimSuffix(name, suffix)
		if barcode == "" {
			continue
		}

		if i.tapes[barcode] != nil {
			log.Printf("Warning: duplicate tape barcode %s found in inventory file %s, ignoring (maybe deprecated files?)", barcode, name)
			continue
		}

		log.Printf("Loading tape inventory file with suffix %s: %s", suffix, name)
		tape, err := loader(i, name)
		if err != nil {
			log.Printf("Failed to load tape inventory from %s: %v", name, err)
			continue
		}
		if tape.Barcode != barcode {
			log.Printf("Warning: tape barcode in file %s (%s) does not match filename, ignoring", name, tape.Barcode)
			continue
		}
		i.tapes[tape.Barcode] = tape
	}
}

func (i *Inventory) Reload() error {
	i.tapes = make(map[string]*Tape)
	files, err := os.ReadDir(i.path)
	if err != nil {
		return err
	}

	i.loadTapeList(".proto", files, loadFromFileProto)
	i.loadTapeList(".json", files, loadFromFileJSON)

	return nil
}

func (i *Inventory) GetOrCreateTape(barcode string) *Tape {
	tape := i.tapes[barcode]
	if tape != nil {
		return tape
	}

	tape = &Tape{
		inventory: i,
		Barcode:   barcode,
		Files:     make(map[string]*File),
	}
	i.tapes[barcode] = tape
	return tape
}

func (i *Inventory) GetTapes() map[string]*Tape {
	return i.tapes
}

func (i *Inventory) GetTapesSortByFreeDesc() []*Tape {
	tapes := make([]*Tape, 0, len(i.tapes))
	for _, tape := range i.tapes {
		tapes = append(tapes, tape)
	}
	slices.SortFunc(tapes, func(a, b *Tape) int {
		return int(b.Free) - int(a.Free)
	})
	return tapes
}

func (i *Inventory) GetBestFiles(pathCryptor *encryption.PathCryptor) map[string]*File {
	files := make(map[string]*File)
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

func (i *Inventory) Save() error {
	for _, tape := range i.tapes {
		if tape.Size == 0 {
			continue
		}

		if err := tape.Save(); err != nil {
			return err
		}
	}
	return nil
}
