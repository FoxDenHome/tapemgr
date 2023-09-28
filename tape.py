from os import scandir, path
from stat import S_ISDIR, S_ISREG
from dataclasses import dataclass
from changer import Changer
from drive import Drive
from util import logged_check_output



@dataclass
class FileInfo:
    size: int
    mtime: float

    def is_better_than(self, other: 'FileInfo'):
        return self.mtime > other.mtime

class Tape:
    def __init__(self, barcode: str):
        super().__init__()
        self.barcode = barcode
        self.files: dict[str, FileInfo] = {}
        self.size = 0
        self.free = 0

    def verify_in_changer(self, changer: Changer):
        found_barcode = changer.read_barcode()
        if found_barcode != self.barcode:
            raise ValueError('Could not change to tape "%s", got "%s"!' % (self.barcode, found_barcode))

    def read_data(self, changer: Changer, drive: Drive, mountpoint: str, readfiles:bool=True):
        if not drive.is_mounter(self):
            self.verify_in_changer(changer)
            did_mount = drive.mount(self, mountpoint)
        else:
            did_mount = False

        line = logged_check_output(['df', '-B1', mountpoint]).split('\n')[1]
        _, size, _, free, _, _ = line.split()
        self.size = int(size)
        self.free = int(free)

        if readfiles:
            self.files = {}
            dir_recurse(mountpoint, self, len(mountpoint))

        if did_mount:
            drive.unmount()

def dir_recurse(dir: str, tape: Tape, mountpoint_len: int):
    for file in scandir(dir):
            stat = file.stat(follow_symlinks=False)
            if S_ISDIR(stat.st_mode):
                dir_recurse(file.path, tape, mountpoint_len)
            elif S_ISREG(stat.st_mode):
                name = path.abspath(file.path)[mountpoint_len:]
                tape.files[name] = FileInfo(size=stat.st_size, mtime=stat.st_mtime)
