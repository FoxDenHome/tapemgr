package descriptor

import "github.com/FoxDenHome/tapemgr/util"

type MediumTransport struct {
	Base

	Except bool
	Full   bool
}

func parseMediumTransport(data []byte, base *Base) (Interface, error) {
	return &MediumTransport{
		Base: *base,

		Except: util.FlagToBool(data[2], 2),
		Full:   util.FlagToBool(data[2], 0),
	}, nil
}
