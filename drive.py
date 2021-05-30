from subprocess import check_call, check_output, Popen
from os.path import ismount
from time import sleep

class Drive:
    def __init__(self, dev):
        self.dev = '/dev/%s' % dev
        self.mountpoint = None
        self.ltfs_process = None

        fh = open('/sys/class/scsi_tape/%s/device/generic/dev' % dev, 'r')
        fd = fh.read().strip()
        fh.close()
        self.sgid = fd.split(':')[1]

    def set_encryption(self, on):
        check_call(['stenc', '-f', self.dev, '-e', 'on' if on else 'off', '-k', '/mnt/keydisk/tape.key', '-a', '1', '--ckod'])

    def eject(self):
        self.unmount()
        check_call(['/opt/tape/TapeTool.sh', 'eject', self.sgid])

    def load(self):
        self.unmount()
        check_call(['/opt/tape/TapeTool.sh', 'load', self.sgid])

    def read_label(self):
        return check_output(['lto-cm', '-f', self.dev, '-r', '2051'], timeout=5).decode().strip()

    def format(self, label, barcode):
        self.unmount()
        self.load()
        self.set_encryption(True)
        check_call(['mkltfs', '--device=%s' % self.dev, '-n', label, '-s', barcode, '-f'])

    def mount(self, mountpoint):
        if self.mountpoint == mountpoint:
            return False
        self.unmount()
        self.load()
        self.set_encryption(True)
        self.mountpoint = mountpoint
        self.ltfs_process = Popen(['ltfs', '-f', '-o', 'umask=077', '-o', 'eject', '-o', 'sync_type=unmount', mountpoint])
        while self.ltfs_process.returncode is None:
            if ismount(mountpoint):
                break
            sleep(0.1)
        if not ismount(mountpoint):
            raise SystemError("Could not mount LTFS tape!")
        return True

    def unmount(self):
        if self.mountpoint is None:
            return False
        check_call(['umount', self.mountpoint])
        self.ltfs_process.wait()
        self.ltfs_process = None
        self.mountpoint = None
        return True
