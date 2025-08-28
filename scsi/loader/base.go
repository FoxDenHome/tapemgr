package loader

const LOADER_MAX_ELEMENTS = 255

type TapeLoader struct {
	DevicePath string
}

func NewTapeLoader(devicePath string) (*TapeLoader, error) {
	return &TapeLoader{
		DevicePath: devicePath,
	}, nil
}
