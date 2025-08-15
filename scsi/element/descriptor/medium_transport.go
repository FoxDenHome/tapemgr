package descriptor

import (
	"errors"
)

type MediumTransport struct {
	Base
}

func parseMediumTransport(data []byte, hasVolTag bool, base *Base) (Interface, error) {
	if len(data) < 2 {
		return nil, errors.New("invalid data length")
	}

	return &MediumTransport{
		Base: *base,
	}, nil
}
