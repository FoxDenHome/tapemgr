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
	tapes map[string]*tape
}

func New(path string) (*Inventory, error) {
	inv := &Inventory{
		path:  path,
		tapes: make(map[string]*tape),
	}
	return inv, inv.Reload()
}

func (i *Inventory) loadTapeList(suffix string, files []os.DirEntry, resave bool, loader func(i *Inventory, filename string) (*tape, error)) {
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

		if resave {
			err = tape.save()
			if err != nil {
				log.Printf("Failed to re-save tape inventory for %s: %v", tape.Barcode, err)
			}
		}
	}
}

func (i *Inventory) Reload() error {
	i.tapes = make(map[string]*tape)
	files, err := os.ReadDir(i.path)
	if err != nil {
		return err
	}

	i.loadTapeList(".proto", files, false, loadFromFileProto)

	return nil
}

func (i *Inventory) GetOrCreateTape(barcode string) Tape {
	tp := i.tapes[barcode]
	if tp != nil {
		return tp
	}

	tp = &tape{
		inventory: i,
		ProtoTape: ProtoTape{
			Barcode: barcode,
			Files:   make(map[string]*ProtoFile),
		},
	}
	i.tapes[barcode] = tp
	return tp
}

func (i *Inventory) HasTape(barcode string) bool {
	return i.tapes[barcode] != nil
}

func (i *Inventory) TapeCount() int {
	return len(i.tapes)
}

func (i *Inventory) GetTapesSortByFreeDesc() []Tape {
	tapes := make([]Tape, 0, len(i.tapes))
	for _, tape := range i.tapes {
		tapes = append(tapes, tape)
	}
	slices.SortFunc(tapes, func(a, b Tape) int {
		return int(b.GetFree()) - int(a.GetFree())
	})
	return tapes
}

func (i *Inventory) GetBestFiles(pathCryptor *encryption.PathCryptor) map[string]File {
	files := make(map[string]File)
	for _, tape := range i.tapes {
		for path, protoFile := range tape.Files {
			clearName, err := pathCryptor.Decrypt(path)
			if err != nil {
				log.Printf("failed to decrypt path %q: %v", path, err)
				continue
			}
			oldInfo, ok := files[clearName]
			if !ok || protoFile.IsBetterThan(oldInfo.(*file).ProtoFile) {
				files[clearName] = &file{
					ProtoFile: protoFile,
					tape:      tape,
					path:      path,
				}
			}
		}
	}

	for name, file := range files {
		if file.GetSize() <= 0 {
			delete(files, name)
		}
	}

	return files
}
