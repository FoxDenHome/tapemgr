package inventory

import (
	"os"
	"strings"
)

type Tape struct {
	inventory *Inventory

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
		tape, err := LoadFromFile(i, name)
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
	tape.inventory = i
	i.tapes[tape.Barcode] = tape
	return tape.Save()
}

func (i *Inventory) Save() error {
	for _, tape := range i.tapes {
		if err := tape.Save(); err != nil {
			return err
		}
	}
	return nil
}
