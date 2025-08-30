package inventory

import (
	"log"
	"os"
	"slices"
	"strings"

	"github.com/FoxDenHome/tapemgr/storage/encryption"
)

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

func (i *Inventory) Reload() error {
	i.tapes = make(map[string]*Tape)
	files, err := os.ReadDir(i.path)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		tape, err := LoadFromFile(i, name)
		if err != nil {
			return err
		}
		i.tapes[tape.Barcode] = tape
	}
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
