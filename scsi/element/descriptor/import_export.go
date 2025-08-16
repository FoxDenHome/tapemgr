package descriptor

import "github.com/FoxDenHome/tapemgr/util"

type ImportExport struct {
	Base

	OIR     bool
	CMC     bool
	InEncab bool
	ExEncab bool
	Access  bool
	ImpExp  bool
	Except  bool
	Full    bool
}

func parseImportExport(data []byte, base *Base) (Interface, error) {
	return &ImportExport{
		Base:    *base,
		OIR:     util.FlagToBool(data[2], 7),
		CMC:     util.FlagToBool(data[2], 6),
		InEncab: util.FlagToBool(data[2], 5),
		ExEncab: util.FlagToBool(data[2], 4),
		Access:  util.FlagToBool(data[2], 3),
		Except:  util.FlagToBool(data[2], 2),
		ImpExp:  util.FlagToBool(data[2], 1),
		Full:    util.FlagToBool(data[2], 0),
	}, nil
}
