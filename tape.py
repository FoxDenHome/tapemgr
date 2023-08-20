from os import scandir, path
from stat import S_ISDIR, S_ISREG
from dataclasses import dataclass
from storage import save_tape
from util import logged_check_output

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
    def __init__(self, barcode):
        self.barcode = barcode
        self.files = {}
        self.size = 0
        self.free = 0

    def verify_in_changer(self, changer):
        found_barcode = changer.read_barcode()
        if found_barcode != self.barcode:
            raise ValueError('Could not change to tape "%s", got "%s"!' % (self.barcode, found_barcode))

    def read_data(self, changer, drive, mountpoint, readfiles=True):
        self.verify_in_changer(changer)
        did_mount = drive.mount(mountpoint)

        line = logged_check_output(['df', '-B1', mountpoint]).decode().split('\n')[1]
        _, size, _, free, _, _ = line.split()
        self.size = int(size)
        self.free = int(free)

        if readfiles:
            self.files = {}
            dir_recurse(mountpoint, self, len(mountpoint))

        save_tape(self)

        if did_mount:
            drive.unmount()
