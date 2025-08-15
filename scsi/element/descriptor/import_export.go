package descriptor

import (
	"errors"
)

type ImportExport struct {
	Base
}

func parseImportExport(data []byte, hasVolTag bool, base *Base) (Interface, error) {
	if len(data) < 2 {
		return nil, errors.New("invalid data length")
	}

	return &ImportExport{
		Base: *base,
	}, nil
}
