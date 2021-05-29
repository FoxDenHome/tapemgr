from subprocess import check_output

class Tape():
    def __init__(self, label):
        self.label = label
        self.files = set()
        self.size = 0
        self.free = 0

    def verify_in_drive(self, drive):
        if drive.read_label() != self.label:
            raise ValueError('Please insert tape "%s" into drive!' % self.label)

    def read_data(self, drive, mountpoint='/mnt/tape'):
        self.verify_in_drive(drive)
        did_mount = drive.mount(mountpoint)

        line = check_output('df', '-B1', mountpoint).split('\n')[1]
        _, size, _, free = line.split()
        self.size = int(size)
        self.free = int(free)

        # TODO: Read files
        self.files = set()

        if did_mount:
            drive.unmount()
