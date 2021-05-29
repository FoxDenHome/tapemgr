from subprocess import call, check_output
from time import sleep

class Drive:
    def __init__(self, dev):
        self.dev = '/dev/%s' % dev
        self.mountpoint = None

        fh = open('/sys/class/scsi_tape/%s/device/generic/dev' % dev, 'r')
        fd = fh.read().strip()
        fh.close()
        self.sgid = fd.split(':')[1]

    def set_encryption(self, on):
        call(['stenc', '-f', self.dev, '-e', 'on' if on else 'off', '-k', '/mnt/keydisk/tape.key', '-a', '1', '--ckod'])

    def eject(self):
        self.unmount()
        call(['/opt/tape/TapeTool.sh', 'eject', self.sgid])

    def load(self):
        self.unmount()
        call(['/opt/tape/TapeTool.sh', 'load', self.sgid])

    def read_label(self):
        return check_output(['lto-cm', '-f', self.dev, '-r', '2051'], timeout=5).decode().strip()

    def format(self, label):
        self.unmount()
        self.load()
        self.set_encryption(True)
        call(['mkltfs', '--device=%s' % self.dev, '-n', label, '-s', label, '-f'])

    def mount(self, mountpoint):
        if self.mountpoint == mountpoint:
            return False
        self.unmount()
        self.load()
        self.set_encryption(True)
        self.mountpoint = mountpoint
        call(['ltfs', '-o', 'umask=077', '-o', 'eject', '-o', 'sync_type=unmount', mountpoint])
        return True

    def unmount(self):
        if self.mountpoint is None:
            return False
        call(['umount', self.mountpoint])
        self.mountpoint = None
        sleep(60)
        return True
