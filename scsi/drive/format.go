package drive

import (
	"os"
	"os/exec"
)

func (d *TapeDrive) Format(barcode string) error {
	if d.isMounted() {
		return ErrAlreadyMounted
	}

	serial := barcode[:6]

	cmd := exec.Command("mkltfs", "--device", d.DevicePath, "-n", barcode, "-s", serial, "-f")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
