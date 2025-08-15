package descriptor

import (
	"errors"
)

type Storage struct {
	Base
}

func parseStorage(data []byte, hasVolTag bool, base *Base) (Interface, error) {
	if len(data) < 2 {
		return nil, errors.New("invalid data length")
	}

	return &Storage{
		Base: *base,
	}, nil
}
