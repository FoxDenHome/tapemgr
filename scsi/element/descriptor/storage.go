package descriptor

import "github.com/FoxDenHome/tapemgr/util"

type Storage struct {
	Base

	Access bool
	Except bool
	Full   bool
}

func parseStorage(data []byte, base *Base) (Interface, error) {
	return &Storage{
		Base:   *base,
		Access: util.FlagToBool(data[2], 3),
		Except: util.FlagToBool(data[2], 2),
		Full:   util.FlagToBool(data[2], 0),
	}, nil
}
