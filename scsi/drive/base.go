package drive

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/FoxDenHome/tapemgr/scsi"
)

var ErrAlreadyMounted = errors.New("tape drive is already mounted")
var ErrLTFSExited = errors.New("ltfs command exited unexpectedly")

type TapeDrive struct {
	DevicePath  string
	GenericPath string

	mountPoint string
	mountWait  *sync.WaitGroup
	mountProc  *exec.Cmd
}

func NewTapeDrive(devicePath string, mountPoint string) (*TapeDrive, error) {
	realDevicePath, err := filepath.EvalSymlinks(devicePath)
	if err != nil {
		return nil, err
	}

	devName := strings.TrimPrefix(realDevicePath, "/dev/")
	genericLink := fmt.Sprintf("/sys/class/scsi_tape/%s/device/generic", devName)

	linkDest, err := os.Readlink(genericLink)
	if err != nil {
		return nil, err
	}

	return &TapeDrive{
		DevicePath:  devicePath,
		GenericPath: fmt.Sprintf("/dev/%s", filepath.Base(linkDest)),
		mountPoint:  mountPoint,
	}, nil
}

func (d *TapeDrive) SerialNumber() (string, error) {
	dev, err := scsi.Open(d.GenericPath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = dev.Close()
	}()

	return dev.SerialNumber()
}

func (d *TapeDrive) MountPoint() string {
	return d.mountPoint
}
