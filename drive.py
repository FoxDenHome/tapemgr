from subprocess import Popen
from os.path import ismount, basename
from time import sleep
from os import readlink
from util import logged_check_call, resolve_symlink
from typing import Any

class Drive:
    mountpoint: str | None
    ltfs_process: Popen[bytes] | None
    mounter: object

    def __init__(self, dev: str):
        super().__init__()
        self.dev = resolve_symlink(dev)
        self.mountpoint = None
        self.ltfs_process = None
        self.mounter = None

    def load(self):
        self.unmount()
        logged_check_call(['sg_start', '--load', self.dev])

    def format(self, label: str, serial: str):
        self.unmount()
        self.load()
        logged_check_call(['mkltfs', '--device=%s' % self.dev, '-n', label, '-s', serial, '-f'])

    def make_sg(self):
        linkdest = readlink('/sys/class/scsi_tape/%s/device/generic' % self.dev.replace('/dev/', ''))
        return '/dev/%s' % basename(linkdest)

    def is_mounter(self, source: Any) -> bool:
        return self.mounter == source

    def mount(self, source: Any, mountpoint: str):
        if self.mountpoint == mountpoint:
            if self.mounter != source:
                raise ValueError('Mountpoint already mounted by different source!')
            return False
        self.unmount()
        self.load()
        self.mountpoint = mountpoint
        self.mounter = source
        self.ltfs_process = Popen(['ltfs', '-o', 'devname=%s' % self.make_sg(), '-f', '-o', 'umask=077', '-o', 'eject', '-o', 'sync_type=unmount', mountpoint])
        while self.ltfs_process.returncode is None:
            if ismount(mountpoint):
                break
            sleep(0.1)
        if not ismount(mountpoint):
            raise SystemError('Could not mount LTFS tape!')
        return True

    def unmount(self) -> None:
        if self.ltfs_process is None:
            return
        if self.mountpoint is None:
            return
        logged_check_call(['umount', self.mountpoint])
        _ = self.ltfs_process.wait()
        self.ltfs_process = None
        self.mountpoint = None
        self.mounter = None
