from os import scandir, path
from stat import S_ISDIR, S_ISREG
from dataclasses import dataclass
from changer import Changer
from drive import Drive
from util import logged_check_output, is_dry_run
from xattr import getxattr
from typing import cast, Any



@dataclass
class FileInfo:
    size: int
    mtime: float
    partition: str = ''
    startblock: int = -1

    def getxattr(self, name: str) -> None:
        try:
            self.partition = cast(bytes, getxattr(name, 'user.ltfs.partition')).decode('utf-8')
        except OSError:
            self.partition = ''
        try:
            self.startblock = int(cast(bytes, getxattr(name, 'user.ltfs.startblock')), 10)
        except OSError:
            self.startblock = -1

    def as_dict(self) -> dict[str, Any]:
        return {
            "size": self.size,
            "mtime": self.mtime,
            "partition": self.partition,
            "start_block": self.startblock
        }

    @staticmethod
    def from_dict(data: dict[str, Any]) -> 'FileInfo':
        return FileInfo(
            size=data["size"],
            mtime=data["mtime"],
            partition=data["partition"],
            startblock=data["start_block"]
        )

    def is_better_than(self, other: 'FileInfo'):
        return self.mtime > other.mtime

class Tape:
    barcode: str
    files: dict[str, FileInfo]
    size: int
    free: int

    def __init__(self, barcode: str):
        super().__init__()
        self.barcode = barcode
        self.files: dict[str, FileInfo] = {}
        self.size = 0
        self.free = 0

    def as_dict(self) -> dict[str, Any]:
        return {
            "barcode": self.barcode,
            "files": {k: v.as_dict() for k, v in self.files.items()},
            "size": self.size,
            "free": self.free
        }

    @staticmethod
    def from_dict(data: dict[str, Any]) -> 'Tape':
        tape = Tape(barcode=data["barcode"])
        tape.files = {k: FileInfo.from_dict(v) for k, v in data["files"].items()}
        tape.size = data["size"]
        tape.free = data["free"]
        return tape

    def verify_in_changer(self, changer: Changer):
        if is_dry_run():
            return

        found_barcode = changer.read_barcode()
        if found_barcode != self.barcode:
            raise ValueError('Could not change to tape "%s", got "%s"!' % (self.barcode, found_barcode))

    def read_data(self, changer: Changer, drive: Drive, mountpoint: str, readfiles:bool=True):
        if not drive.is_mounter(self):
            self.verify_in_changer(changer)
            did_mount = drive.mount(self, mountpoint)
        else:
            did_mount = False

        if is_dry_run():
            if did_mount:
                drive.unmount()
            return

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
            file_info = FileInfo(size=stat.st_size, mtime=stat.st_mtime)
            file_info.getxattr(file.path)
            tape.files[name] = file_info
