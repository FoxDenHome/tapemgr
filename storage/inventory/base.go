package inventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Tape struct {
	Barcode string               `json:"barcode"`
	Files   map[string]*FileInfo `json:"files"`
	Size    int64                `json:"size"`
	Free    int64                `json:"free"`
}

type Inventory struct {
	path  string
	tapes map[string]*Tape
}

func New() (*Inventory, error) {
	inv := &Inventory{
		path:  "tapes",
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
		tape, err := LoadFromFile(filepath.Join(i.path, name))
		if err != nil {
			return err
		}
		i.tapes[tape.Barcode] = tape
	}
	return nil
}

func (i *Inventory) AddTape(tape *Tape) error {
	if tape == nil || tape.Barcode == "" {
		return nil
	}
	i.tapes[tape.Barcode] = tape
	return i.SaveTape(tape.Barcode)
}

func (i *Inventory) Save() error {
	for barcode := range i.tapes {
		if err := i.SaveTape(barcode); err != nil {
			return err
		}
	}
	return nil
}

func (i *Inventory) SaveTape(barcode string) error {
	tape, ok := i.tapes[barcode]
	if !ok {
		return nil // Tape not found, nothing to save
	}

	fh, err := os.Create(filepath.Join(i.path, barcode+".json"))
	if err != nil {
		return err
	}
	defer fh.Close()

	enc := json.NewEncoder(fh)
	return enc.Encode(tape)
}
