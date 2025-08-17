package drive

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

type TapeDrive struct {
	DevicePath  string
	GenericPath string

	mountPoint string
	mountProc  *exec.Cmd
}

func NewTapeDrive(devicePath string, mountPoint string) (*TapeDrive, error) {
	devName := strings.TrimPrefix(devicePath, "/dev/")
	genericLink := fmt.Sprintf("/sys/class/scsi_Tape/%s/device/generic", devName)

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

func (d *TapeDrive) Mount() error {
	if isMount(d.mountPoint) {
		return nil
	}

	err := d.Unmount()
	if err != nil {
		return err
	}

	d.mountProc = exec.Command("ltfs", "-o", "devname="+d.GenericPath, "-f", "-o", "umask=077", "-o", "eject", "-o", "sync_type=unmount", d.mountPoint)
	d.mountProc.Stdout = os.Stdout
	d.mountProc.Stderr = os.Stderr

	err = d.mountProc.Start()
	if err != nil {
		return fmt.Errorf("error mounting tape drive: %v", err)
	}

	for {
		if syscall.Kill(d.mountProc.Process.Pid, 0) != nil {
			return errors.New("ltfs command exited unexpectedly")
		}

		if isMount(d.mountPoint) {
			return nil
		}
	}
}

func (d *TapeDrive) Unmount() error {
	if d.mountProc == nil {
		return nil
	}

	err := syscall.Unmount(d.mountPoint, 0)
	if err != nil {
		return err
	}

	proc := d.mountProc
	d.mountProc = nil
	return proc.Wait()
}

func (d *TapeDrive) Barcode() string {
	return "TODO"
}

func (d *TapeDrive) MountPoint() string {
	return d.mountPoint
}

func isMount(path string) bool {
	mounts, err := os.ReadFile("/proc/self/mounts")
	if err != nil {
		return false
	}

	for _, line := range strings.Split(string(mounts), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		if fields[1] == path {
			return true
		}
	}

	return false
}
