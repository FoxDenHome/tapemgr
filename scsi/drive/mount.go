package drive

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func (d *TapeDrive) Mount() error {
	if d.isMounted() {
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
		return err
	}

	d.mountWait.Add(1)
	go func() {
		_ = d.mountProc.Wait()
		d.mountWait.Done()
	}()

	for {
		if !d.mountProcAlive() {
			return ErrLTFSExited
		}

		if d.isMounted() {
			return nil
		}
	}
}

func (d *TapeDrive) Unmount() (err error) {
	if d.mountProc == nil {
		return nil
	}

	proc := d.mountProc
	d.mountProc = nil

	if d.isMounted() {
		err = syscall.Unmount(d.mountPoint, 0)
	} else if d.mountProcAlive() {
		err = d.mountProc.Process.Kill()
	} else {
		return nil
	}

	if err != nil {
		return
	}

	_ = proc.Wait()
	return nil
}

func (d *TapeDrive) WaitForUnmount() {
	d.mountWait.Wait()
}

func (d *TapeDrive) mountProcAlive() bool {
	return d.mountProc.ProcessState == nil
}

func (d *TapeDrive) isMounted() bool {
	mounts, err := os.ReadFile("/proc/self/mounts")
	if err != nil {
		panic(err)
	}

	for _, line := range strings.Split(string(mounts), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		if fields[1] == d.mountPoint {
			return true
		}
	}

	return false
}
