package descriptor

import (
	"errors"
)

type DataTransfer struct {
	Base
}

func parseDataTransfer(data []byte, hasVolTag bool, base *Base) (Interface, error) {
	if len(data) < 2 {
		return nil, errors.New("invalid data length")
	}

	return &DataTransfer{
		Base: *base,
	}, nil
}
