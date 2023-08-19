from subprocess import check_call, check_output, Popen
from os.path import ismount, basename
from time import sleep
from os import readlink

class Drive:
    def __init__(self, dev):
        self.dev = dev
        self.mountpoint = None
        self.ltfs_process = None

    def set_encryption(self, on):
        check_call(['stenc', '-f', self.dev, '-e', 'on' if on else 'off', '-k', '/mnt/keydisk/tape.key', '-a', '1', '--ckod'])

    def load(self):
        self.unmount()
        check_call(['sg_start', '--load', self.dev])

    def read_label(self):
        try:
            return check_output(['lto-cm', '-f', self.dev, '-r'], timeout=5).decode().strip()
        except:
            return None

    def format(self, label, serial):
        self.unmount()
        self.load()
        self.set_encryption(True)
        check_call(['mkltfs', '--device=%s' % self.dev, '-n', label, '-s', serial, '-f'])

    def make_sg(self):
        linkdest = readlink('/sys/class/scsi_tape/%s/device/generic' % self.dev.replace('/dev/', ''))
        return '/dev/%s' % basename(linkdest)

    def mount(self, mountpoint):
        if self.mountpoint == mountpoint:
            return False
        self.unmount()
        self.set_encryption(True)
        self.mountpoint = mountpoint
        self.ltfs_process = Popen(['ltfs', '-o', 'devname=%s' % self.make_sg(), '-f', '-o', 'umask=077', '-o', 'eject', '-o', 'sync_type=unmount', mountpoint])
        while self.ltfs_process.returncode is None:
            if ismount(mountpoint):
                break
            sleep(0.1)
        if not ismount(mountpoint):
            raise SystemError('Could not mount LTFS tape!')
        return True

    def unmount(self):
        if self.mountpoint is None:
            return False
        check_call(['umount', self.mountpoint])
        self.ltfs_process.wait()
        self.ltfs_process = None
        self.mountpoint = None
        return True
