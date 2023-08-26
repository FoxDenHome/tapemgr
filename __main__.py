from dataclasses import dataclass
from types import FrameType
from typing import Literal, cast, overload
from tape import FileInfo, Tape
from drive import Drive
from changer import Changer
from os import path, lstat, listdir, stat_result
from os.path import join as path_join, basename
from stat import S_ISDIR, S_ISREG
from storage import save_tape, load_all_tapes, set_storage_dir
from argparse import ArgumentParser
from signal import SIGINT, SIGTERM, signal
from util import logged_check_call, logged_call, format_size, format_mtime

TAPE_SIZE_SPARE = 1024 * 1024 * 1024 # 1 GB
TAPE_SIZE_NEW_SPARE = 2 * TAPE_SIZE_SPARE

@dataclass
class ArgParseResult:
    mount: str
    tape_prefix: str
    tape_type: str
    tape_dir: str
    tape_suffix: str
    device: str
    tape_key: str
    changer: str
    changer_drive_index: int
    include_hidden: bool
    action: list[str]
    files: list[str]

parser = ArgumentParser(description='Tape manager')
_ = parser.add_argument('action', metavar='action', type=str, nargs=1, help='The action to perform')
_ = parser.add_argument('files', metavar='files', type=str, nargs='*', help='Files to store (for store action)')
_ = parser.add_argument('--device', dest='device', type=str, default='/dev/nst0')
_ = parser.add_argument('--changer', dest='changer', type=str, default='/dev/sch0')
_ = parser.add_argument('--changer-drive-index', dest='changer_drive_index', type=int, default=0)
_ = parser.add_argument('--mount', dest='mount', type=str, default='/mnt/tape')
_ = parser.add_argument('--tape-dir', dest='tape_dir', type=str, default=path.join(path.dirname(__file__), 'tapes'))
_ = parser.add_argument('--tape-prefix', dest='tape_prefix', type=str, default='P', help='Prefix to add to tape label and barcode')
_ = parser.add_argument('--tape-suffix', dest='tape_suffix', type=str, default='S', help='Suffix to add to tape label and barcode')
_ = parser.add_argument('--tape-type', dest='tape_type', type=str, default='L6', help='Tape type (L6 for LTO-6, L7 for LTO-7 etc)')
_ = parser.add_argument('--tape-key', dest='tape_key', type=str, default='/mnt/keydisk/tape.key', help='Tape key file for encryption, blank to disable')
_ = parser.add_argument('--include-hidden', dest='include_hidden', action='store_true', help='Include hidden files in backup (default: false)')

args = cast(ArgParseResult, parser.parse_args())

if len(args.tape_type) != 2 or args.tape_type[0] != 'L':
    raise ValueError('Tape type must be L#')

set_storage_dir(args.tape_dir)
TAPE_MOUNT = args.mount
TAPE_PREFIX = args.tape_prefix
TAPE_SUFFIX = args.tape_suffix
TAPE_TYPE = args.tape_type
TAPE_LABEL_FMT = f'%s%0{6 - (len(TAPE_PREFIX) + len(TAPE_SUFFIX))}d%s'

INCLUDE_HIDDEN = args.include_hidden

current_tape = None

drive = Drive(args.device, args.tape_key)
changer = Changer(args.changer, args.changer_drive_index)
tapes = load_all_tapes()
action = args.action[0]

should_exit = False


def signal_exit_handler(_sig: int, _frame: FrameType | None) -> None:
    global should_exit
    should_exit = True
    print('Got exit signal, exiting ASAP...')
_ = signal(SIGINT, signal_exit_handler)
_ = signal(SIGTERM, signal_exit_handler)

def should_backup_filename(name: str) -> bool:
    if should_exit:
        return False
    if INCLUDE_HIDDEN:
        return True
    return basename(name)[0] != '.'

def save_all_tapes():
    for tape in tapes.values():
        save_tape(tape)

def refresh_current_tape():
    global current_tape
    if not current_tape:
        raise Exception('no tape selected')
    current_tape.read_data(changer, drive, TAPE_MOUNT, False)

def make_tape_barcode():
    idx = 0
    while True:
        idx += 1
        label = TAPE_LABEL_FMT % (TAPE_PREFIX, idx, TAPE_SUFFIX)
        barcode = '%s%s' % (label, TAPE_TYPE)
        if barcode not in tapes:
            return barcode

def load_tape(barcode: str):
    print('Loading tape by barcode "%s"' % barcode)
    changer.load_by_barcode(barcode)

def ask_for_tape(barcode: str | None):
    global current_tape

    if barcode is None:
        changer.unload_current()
        format_current_tape(True)
        return

    while True:
        current_tape = get_current_tape()
        if current_tape and current_tape.barcode == barcode:
            _ = drive.mount(current_tape, TAPE_MOUNT)
            current_tape.read_data(changer, drive, TAPE_MOUNT)
            return
        load_tape(barcode)

@overload
def get_current_tape(create_new:Literal[True]) -> Tape: ...

@overload
def get_current_tape(create_new:bool=False) -> Tape | None: ...

def get_current_tape(create_new:bool=False):
    barcode = changer.read_barcode()
    if barcode is None:
        return None

    if barcode not in tapes:
        if create_new:
            tape = Tape(barcode)
            tapes[barcode] = tape
            return tape
        return None

    return tapes[barcode]

def format_current_tape(mount:bool=False):
    global current_tape

    if get_current_tape():
        raise ValueError('Tape is already in this program!')

    barcode = make_tape_barcode()
    load_tape(barcode)

    drive.format(barcode, barcode[:6])

    tape = Tape(barcode)
    tape.verify_in_changer(changer)
    current_tape = tape
    if mount:
        _ = drive.mount(current_tape, TAPE_MOUNT)
    tape.read_data(changer, drive, TAPE_MOUNT)
    tapes[barcode] = tape
    save_tape(tape)
    print ('Formatted tape with barcode "%s"!' % barcode)

def backup_file(file: str, fstat: stat_result):
    global current_tape

    if not should_backup_filename(file):
        return

    name = path.abspath(file)
    dir = path.dirname(name)

    finfo = FileInfo(size=fstat.st_size,mtime=fstat.st_mtime)

    for tape in tapes.values():
        if name in tape.files and not finfo.is_better_than(tape.files[name]):
            print('[SKIP] %s' % name)
            return

    print('[STOR] %s' % name)

    min_size = fstat.st_size + TAPE_SIZE_SPARE
    min_size_new = fstat.st_size + TAPE_SIZE_NEW_SPARE

    if current_tape is not None and current_tape.free < min_size:
        refresh_current_tape()

    while current_tape is None or current_tape.free < min_size:
        drive.unmount()
        # Find new tape!
        min_size = min_size_new
        found_existing_tape = False
        for barcode, tape in tapes.items():
            if tape.free >= min_size:
                found_existing_tape = True
                ask_for_tape(barcode)
                break
        if not found_existing_tape:
            ask_for_tape(None)
        _ = drive.mount(current_tape, TAPE_MOUNT)
        refresh_current_tape()

    tape_name = '%s%s' % (TAPE_MOUNT, name)

    logged_check_call(['mkdir', '-p', '%s%s' % (TAPE_MOUNT, dir)])
    logged_call(['cp', name, tape_name])

    fstat_tape = lstat(tape_name)
    current_tape.files[name] = FileInfo(size=fstat_tape.st_size,mtime=fstat_tape.st_mtime)

    refresh_current_tape()

def backup_recursive(dir: str):
    if not should_backup_filename(dir):
        return

    for file in listdir(dir):
        fullname = path_join(dir, file)
        stat = lstat(fullname)
        if S_ISDIR(stat.st_mode):
            backup_recursive(fullname)
        elif S_ISREG(stat.st_mode):
            backup_file(fullname, stat)


if action == 'format':
    format_current_tape()
elif action == 'store':
    try:
        for name in args.files:
            stat = lstat(name)
            if S_ISDIR(stat.st_mode):
                backup_recursive(name)
            elif S_ISREG(stat.st_mode):
                backup_file(name, stat)
            else:
                raise ValueError('Cannot backup file (not regular file or directory): %s' % name)
    finally:
        drive.unmount()
        changer.unload_current()
elif action == 'unload':
    changer.unload_current()
elif action == 'index':
    current_tape = get_current_tape(create_new=True)
    current_tape.read_data(changer, drive, TAPE_MOUNT)
elif action == 'list':
    files: dict[str, tuple[FileInfo, Tape]] = {}
    for tape in tapes.values():
        for name, info in tape.files.items():
            if name in files and not info.is_better_than(files[name][0]):
                continue
            files[name] = (info, tape)

    for name, info_tuple in files.items():
        info, tape = info_tuple
        print('[%s] Name "%s", size %s, mtime %s' % (tape.barcode, name, format_size(info.size), format_mtime(info.mtime)))
elif action == 'find':
    best_info: FileInfo | None = None
    best_tape: Tape | None = None
    for tape in tapes.values():
        for name, info in tape.files.items():
            if name != args.files[1]:
                continue
            print('Found copy of file on "%s", size %s, mtime %s' % (tape.barcode, format_size(info.size), format_mtime(info.mtime)))
            if best_info is not None and best_info.is_better_than(info):
                continue
            best_info = info
            best_tape = tape
    if best_tape is not None and best_info is not None:
        print('Best copy of file seems to be on "%s", size %s, mtime %s' % (best_tape.barcode, format_size(best_info.size), format_mtime(best_info.mtime)))
    else:
        print('Could not find that file :(')
elif action == 'mount':
    current_tape = get_current_tape()
    if current_tape is not None:
        _ = drive.mount(current_tape, TAPE_MOUNT)
        print('Mounted "%s" to "%s", run "umount %s" and wait for eject once done!' % (current_tape.barcode, TAPE_MOUNT, TAPE_MOUNT))
    else:
        print('Do not recognize this tape!')
elif action == 'statistics':
    for barcode, tape in tapes.items():
        print('[%s] Free = %s / %s (%.2f%%), Files = %d' % (barcode, format_size(tape.free), format_size(tape.size), (tape.free / tape.size) * 100.0, len(tape.files)))
