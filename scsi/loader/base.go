package loader

type TapeLoader struct {
	DevicePath string
}

func NewTapeLoader(devicePath string) (*TapeLoader, error) {
	return &TapeLoader{
		DevicePath: devicePath,
	}, nil
}
