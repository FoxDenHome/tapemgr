package drive

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/FoxDenHome/tapemgr/scsi"
)

var ErrAlreadyMounted = errors.New("tape drive is already mounted")
var ErrLTFSExited = errors.New("ltfs command exited unexpectedly")

type TapeDrive struct {
	DevicePath  string
	GenericPath string

	mountPoint string
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

func (d *TapeDrive) Load() error {
	return d.loadUnload(scsi.LOAD_AND_THREAD)
}

func (d *TapeDrive) Unload() error {
	return d.loadUnload(scsi.UNLOAD_ARCHIVE)
}

func (d *TapeDrive) MountPoint() string {
	return d.mountPoint
}

func (d *TapeDrive) loadUnload(op scsi.LoadUnloadOperation) error {
	if d.isMounted() {
		return ErrAlreadyMounted
	}

	dev, err := scsi.Open(d.DevicePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = dev.Close()
	}()

	err = dev.WaitForReady()
	if err != nil {
		return err
	}
	err = dev.LoadUnload(op)
	if err != nil {
		return err
	}
	err = dev.WaitForReady()
	if err != nil {
		return err
	}
	return nil
}
