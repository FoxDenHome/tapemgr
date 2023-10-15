from drive import Drive
from changer import Changer
from storage import Storage
from tape import FileInfo, Tape
from typing import Optional, Literal, overload
from os.path import basename, join as path_join, abspath, dirname
from os import stat_result, lstat, listdir
from util import logged_call, logged_check_call
from stat import S_ISDIR, S_ISREG
from time import sleep
from util import format_size, format_mtime
from name_enc import NameCryptor

TAPE_SIZE_SPARE = 1024 * 1024 * 1024 # 1 GB
TAPE_SIZE_NEW_SPARE = 2 * TAPE_SIZE_SPARE

class Manager:
    mountpoint: str = '/mnt/tape'
    tape_barcode_prefix: str = ''
    tape_barcode_suffix: str = ''
    tape_type: str = ''
    tape_barcode_fmt: str = ''

    should_exit: bool = False
    include_hidden: bool = False

    drive: Drive
    changer: Changer
    storage: Storage

    current_tape: Optional[Tape] = None

    in_backup: int = 0
    age_recipient_file: str

    def __init__(self, drive: Drive, changer: Changer, storage: Storage, age_recipient_file: str, filename_key_file: str) -> None:
        super().__init__()
        self.drive = drive
        self.changer = changer
        self.storage = storage
    
        self.age_recipient_file = age_recipient_file
        with open(filename_key_file, 'rb') as f:
            self.name_crypto = NameCryptor(f.read())

        self.set_barcode('P', 'S', 'L6')

    def set_barcode(self, prefix: str, suffix: str, type: str) -> None:
        self.tape_barcode_prefix = prefix
        self.tape_barcode_suffix = suffix
        self.tape_type = type
        self.tape_barcode_fmt = f'%s%0{6 - (len(self.tape_barcode_prefix) + len(self.tape_barcode_suffix))}d%s%s'

    def refresh_current_tape(self, readfiles: bool = False):
        if not self.current_tape:
            raise Exception('no tape selected')
        self.current_tape.read_data(self.changer, self.drive, self.mountpoint, readfiles)
        self.storage.save(self.current_tape)

    def make_tape_barcode(self):
        idx = 0
        while True:
            idx += 1
            barcode = self.tape_barcode_fmt % (self.tape_barcode_prefix, idx, self.tape_barcode_suffix, self.tape_type)
            if barcode not in self.storage.tapes:
                return barcode

    def load_tape(self, barcode: str):
        print('Loading tape by barcode "%s"' % barcode)
        self.changer.load_by_barcode(barcode)

    def ask_for_tape(self, barcode: str | None):
        if barcode is None:
            self.changer.unload_current()
            self.format_current_tape(True)
            return

        while True:
            self.current_tape = self.get_current_tape()
            if self.current_tape and self.current_tape.barcode == barcode:
                _ = self.drive.mount(self.current_tape, self.mountpoint)
                self.refresh_current_tape(True)
                return
            self.load_tape(barcode)

    @overload
    def get_current_tape(self, create_new:Literal[True]) -> Tape: ...

    @overload
    def get_current_tape(self, create_new:bool=False) -> Tape | None: ...

    def get_current_tape(self, create_new:bool=False):
        barcode = self.changer.read_barcode()
        if not barcode:
            return None

        if barcode not in self.storage.tapes:
            if create_new:
                tape = Tape(barcode)
                self.storage.tapes[barcode] = tape
                return tape
            return None

        return self.storage.tapes[barcode]

    def should_backup_filename(self, name: str) -> bool:
        if self.should_exit:
            return False
        if self.include_hidden:
            return True
        return basename(name)[0] != '.'

    def format_current_tape(self, mount:bool=False):
        if self.get_current_tape():
            raise ValueError('Tape is already in this program!')

        barcode = self.make_tape_barcode()
        self.load_tape(barcode)

        self.drive.format(barcode, barcode[:6])

        tape = Tape(barcode)
        tape.verify_in_changer(self.changer)
        self.current_tape = tape
        if mount:
            _ = self.drive.mount(self.current_tape, self.mountpoint)
        self.refresh_current_tape(True)
        print ('Formatted tape with barcode "%s"!' % barcode)

    def backup_file(self, file: str, fstat: stat_result):
        if not self.should_backup_filename(file):
            return

        self.in_backup += 1
        try:
            name = abspath(file)

            encrypted_name = self.name_crypto.encrypt(name)

            finfo = FileInfo(size=fstat.st_size,mtime=fstat.st_mtime)

            for tape in self.storage.tapes.values():
                if encrypted_name in tape.files and not finfo.is_better_than(tape.files[encrypted_name]):
                    print('[SKIP] %s' % name)
                    return

            print('[STOR] %s' % name)

            min_size = fstat.st_size + TAPE_SIZE_SPARE
            min_size_new = fstat.st_size + TAPE_SIZE_NEW_SPARE

            if self.current_tape is not None and self.current_tape.free < min_size:
                self.refresh_current_tape()

            while self.current_tape is None or self.current_tape.free < min_size:
                self.drive.unmount()
                # Find new tape!
                min_size = min_size_new
                found_existing_tape = False
                for _, tape in sorted(self.storage.tapes.items()):
                    if tape.free >= min_size:
                        found_existing_tape = True
                        self.ask_for_tape(tape.barcode)
                        break
                if not found_existing_tape:
                    self.ask_for_tape(None)
                _ = self.drive.mount(self.current_tape, self.mountpoint)
                self.refresh_current_tape()

            if encrypted_name[0] != '/':
                encrypted_name = '/' + encrypted_name
            tape_name = '%s%s' % (self.mountpoint, encrypted_name)

            logged_check_call(['mkdir', '-p', dirname(tape_name)])
            logged_call(['age', '-e', '-o', tape_name, '-R', self.age_recipient_file, name])
            logged_call(['touch', '-r', name, tape_name])

            fstat_tape = lstat(tape_name)
            self.current_tape.files[encrypted_name] = FileInfo(size=fstat_tape.st_size,mtime=fstat_tape.st_mtime)

            self.refresh_current_tape()
        finally:
            self.in_backup -= 1

    def backup_recursive(self, dir: str):
        if not self.should_backup_filename(dir):
            return

        self.in_backup += 1
        try:
            for file in listdir(dir):
                fullname = path_join(dir, file)
                stat = lstat(fullname)
                if S_ISDIR(stat.st_mode):
                    self.backup_recursive(fullname)
                elif S_ISREG(stat.st_mode):
                    self.backup_file(fullname, stat)
        finally:
            self.in_backup -= 1

    def shutdown(self) -> None:
        self.should_exit = True

        while self.in_backup:
            sleep(0.1)

        try:
            self.refresh_current_tape(True)
        except:
            pass
        self.drive.unmount()
        self.changer.unload_current()

    def mount(self, barcode: str) -> None:
        self.changer.load_by_barcode(barcode)
        self.current_tape = self.get_current_tape()
        if self.current_tape is not None:
            _ = self.drive.mount(self.current_tape, self.mountpoint)
            print('Mounted "%s" to "%s", run "umount %s" and wait for eject once done!' % (self.current_tape.barcode, self.mountpoint, self.mountpoint))
        else:
            raise ValueError('Do not recognize this tape!')

    def find(self, name: str) -> tuple[FileInfo, Tape]:
        encrypted_name = self.name_crypto.encrypt(name)
    
        best_info: FileInfo | None = None
        best_tape: Tape | None = None
        for tape in self.storage.tapes.values():
            for name, info in tape.files.items():
                if name != encrypted_name:
                    continue
                print('Found copy of file on "%s", size %s, mtime %s' % (tape.barcode, format_size(info.size), format_mtime(info.mtime)))
                if best_info is not None and best_info.is_better_than(info):
                    continue
                best_info = info
                best_tape = tape
        if best_tape is not None and best_info is not None:
            return best_info, best_tape
        else:
            raise ValueError('Could not find file')

    def list_all_best(self):
        files: dict[str, tuple[FileInfo, Tape]] = {}
        for tape in self.storage.tapes.values():
            for encrypted_name, info in tape.files.items():
                if encrypted_name in files and not info.is_better_than(files[encrypted_name][0]):
                    continue
                files[encrypted_name] = (info, tape)
        return files

    def index_tape(self, barcode: str) -> None:
        self.changer.load_by_barcode(barcode)
        self.current_tape = self.get_current_tape(create_new=True)
        self.refresh_current_tape(True)

    def decrypt_filename(self, name: str) -> str:
        return self.name_crypto.decrypt(name)

    def encrypt_filename(self, name: str) -> str:
        return self.name_crypto.encrypt(name)
