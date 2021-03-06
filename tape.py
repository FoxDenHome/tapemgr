from subprocess import check_output
from os import scandir, path
from stat import S_ISDIR, S_ISREG
from dataclasses import dataclass
from storage import save_tape

def dir_recurse(dir, tape, mountpoint_len):
    for file in scandir(dir):
            stat = file.stat(follow_symlinks=False)
            if S_ISDIR(stat.st_mode):
                dir_recurse(file.path, tape, mountpoint_len)
            elif S_ISREG(stat.st_mode):
                name = path.abspath(file.path)[mountpoint_len:]
                tape.files[name] = FileInfo(size=stat.st_size, mtime=stat.st_mtime)

@dataclass
class FileInfo:
    size: int
    mtime: int

    def is_better_than(self, other):
        if self.mtime == other.mtime:
            return self.size > other.size
        return self.mtime > other.mtime

class Tape:
    def __init__(self, label):
        self.label = label
        self.files = {}
        self.size = 0
        self.free = 0

    def verify_in_drive(self, drive):
        found_label = drive.read_label()
        if found_label != self.label:
            raise ValueError('Please insert tape "%s" into drive (current: %s)!' % (self.label, found_label))

    def read_data(self, drive, mountpoint, readfiles=True):
        self.verify_in_drive(drive)
        did_mount = drive.mount(mountpoint)

        line = check_output(['df', '-B1', mountpoint]).decode().split('\n')[1]
        _, size, _, free, _, _ = line.split()
        self.size = int(size)
        self.free = int(free)

        if readfiles:
            self.files = {}
            dir_recurse(mountpoint, self, len(mountpoint))

        save_tape(self)

        if did_mount:
            drive.unmount()
