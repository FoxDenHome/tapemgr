package inventory

import (
	"encoding/json"
	"os"

	"github.com/FoxDenHome/tapemgr/scsi/drive"
)

func LoadFromFile(filename string) (*Tape, error) {
	fh, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	tape := &Tape{}
	dec := json.NewDecoder(fh)
	err = dec.Decode(tape)
	if err != nil {
		return nil, err
	}
	for name, file := range tape.Files {
		if name[0] == '/' {
			name = name[1:]
			delete(tape.Files, name)
			tape.Files[name] = file
		}
		file.tape = tape
		file.name = name
	}
	return tape, nil
}

func LoadFromDrive(drive *drive.TapeDrive) (*Tape, error) {
	return nil, nil
}

func (i *Inventory) GetTapes() []string {
	var tapes []string
	for _, tape := range i.tapes {
		tapes = append(tapes, tape.Barcode)
	}
	return tapes
}
