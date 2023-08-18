from tape import FileInfo, Tape
from drive import Drive
from changer import Changer
from os import path, lstat, scandir
from subprocess import call, check_call
from stat import S_ISDIR, S_ISREG
from storage import save_tape, load_all_tapes, set_storage_dir
from datetime import datetime
from argparse import ArgumentParser

TAPE_MOUNT = None
TAPE_PREFIX = None
TAPE_SUFFIX = None
TAPE_TYPE = None
TAPE_SIZE_SPARE = 1024 * 1024 * 1024 # 1 GB
TAPE_LABEL_FMT = None

drive = None
changer = None
tapes = {}
current_tape = None

def save_all_tapes():
    for _, tape in tapes.items():
        save_tape(tape)

def refresh_current_tape():
    current_tape.read_data(drive, TAPE_MOUNT, False)

def make_tape_label():
    idx = 0
    while True:
        idx += 1
        label = TAPE_LABEL_FMT % (TAPE_PREFIX, idx, TAPE_SUFFIX)
        if label not in tapes:
            return label

def barcode_from_label(label):
    return '%s%s' % (label, TAPE_TYPE)

def serial_from_label(label):
    return label

def load_tape(label):
    barcode = barcode_from_label(label)
    print('Loading tape label "%s" by barcode "%s"' % (label, barcode))
    changer.load_by_barcode(barcode)

def ask_for_tape(label):
    global current_tape

    if label is None:
        changer.eject()
        new_label = make_tape_label()
        load_tape(new_label)
        format_current_tape(new_label, True)
        return

    while True:
        current_tape = get_current_tape()
        if current_tape and current_tape.label == label:
            drive.mount(TAPE_MOUNT)
            return
        changer.eject()
        load_tape(label)

def get_current_tape(create_new=False):
    label = drive.read_label()
    if label is None:
        #try:
        #    drive.load()
        #except:
        #    return None
        label = drive.read_label()
        if label is None:
            return None

    if not label in tapes:
        if create_new:
            tape = Tape(label)
            tapes[label] = tape
            return tape
        return None

    return tapes[label]

def format_current_tape(label=None, mount=False):
    global current_tape

    if get_current_tape():
        raise ValueError('Tape is already in this program!')
    if label is None:
        label = make_tape_label()
    drive.format(label, serial_from_label(label))

    tape = Tape(label)
    tape.verify_in_drive(drive)
    current_tape = tape
    if mount:
        drive.mount(TAPE_MOUNT)
    tape.read_data(drive, TAPE_MOUNT)
    tapes[label] = tape
    save_tape(tape)
    print ('Formatted tape with label "%s"!' % label)

def backup_file(file):
    global current_tape

    name = path.abspath(file.path)
    dir = path.dirname(name)

    fstat = file.stat(follow_symlinks=False)
    finfo = FileInfo(size=fstat.st_size,mtime=fstat.st_mtime)

    for _, tape in tapes.items():
        if name in tape.files and not finfo.is_better_than(tape.files[name]):
            print('[SKIP] %s' % name)
            return

    print('[STOR] %s' % name)

    min_size = fstat.st_size + TAPE_SIZE_SPARE

    if current_tape is not None and current_tape.free < min_size:
        refresh_current_tape()

    while current_tape is None or current_tape.free < min_size:
        drive.unmount()
        # Find new tape!
        found_existing_tape = False
        for label, tape in tapes.items():
            if tape.free >= min_size:
                found_existing_tape = True
                ask_for_tape(label)
                break
        if not found_existing_tape:
            ask_for_tape(None)
        drive.mount(TAPE_MOUNT)
        refresh_current_tape()

    tape_name = '%s%s' % (TAPE_MOUNT, name)

    check_call(['mkdir', '-p', '%s%s' % (TAPE_MOUNT, dir)])
    call(['cp', name, tape_name])

    fstat_tape = lstat(tape_name)
    current_tape.files[name] = FileInfo(size=fstat_tape.st_size,mtime=fstat_tape.st_mtime)

    refresh_current_tape()

def backup_recursive(dir):
    for file in scandir(dir):
            stat = file.stat(follow_symlinks=False)
            if S_ISDIR(stat.st_mode):
                backup_recursive(file.path)
            elif S_ISREG(stat.st_mode):
                backup_file(file)

def format_size(size):
    for unit in ['','Ki','Mi','Gi','Ti','Pi','Ei','Zi']:
        if size < 1024.0:
            return '%3.1f %sB' % (size, unit)
        size /= 1024.0
    return '%.1f %sB' % (size, 'Yi')

def format_mtime(mtime):
    time = datetime.fromtimestamp(mtime)
    return time.strftime("%Y-%m-%d %H:%M:%S")

parser = ArgumentParser(description='Tape manager')
parser.add_argument('action', metavar='action', type=str, nargs=1, help='The action to perform')
parser.add_argument('files', metavar='files', type=str, nargs='*', help='Files to store (for store action)')
parser.add_argument('--device', dest='device', type=str, default='/dev/nst0')
parser.add_argument('--changer', dest='changer', type=str, default='/dev/sch0')
parser.add_argument('--mount', dest='mount', type=str, default='/mnt/tape')
parser.add_argument('--tape-dir', dest='tape_dir', type=str, default=path.join(path.dirname(__file__), 'tapes'))
parser.add_argument('--tape-prefix', dest='tape_prefix', type=str, default='P', help='Prefix to add to tape label and barcode')
parser.add_argument('--tape-suffix', dest='tape_suffix', type=str, default='S', help='Suffix to add to tape label and barcode')
parser.add_argument('--tape-type', dest='tape_type', type=str, default='L6', help='Tape type (L6 for LTO-6, L7 for LTO-7 etc)')

args = parser.parse_args()

if len(args.tape_type) != 2 or args.tape_type[0] != 'L':
    raise ValueError('Tape type must be L#')

set_storage_dir(args.tape_dir)
TAPE_MOUNT = args.mount
TAPE_PREFIX = args.tape_prefix
TAPE_SUFFIX = args.tape_suffix
TAPE_TYPE = args.tape_type
TAPE_LABEL_FMT = f'%s%0{6 - (len(TAPE_PREFIX) + len(TAPE_SUFFIX))}d%s'

drive = Drive(args.device)
changer = Changer(args.changer)
tapes = load_all_tapes()

action = args.action[0]

if action == 'format':
    format_current_tape()
elif action == 'store':
    try:
        for name in args.files:
            stat = lstat(name)
            if S_ISDIR(stat.st_mode):
                backup_recursive(name)
            elif S_ISREG(stat.st_mode):
                backup_file(name)
            else:
                raise ValueError('Cannot backup file (not regular file or directory): %s' % name)
    finally:
        drive.unmount()
elif action == 'index':
    current_tape = get_current_tape(create_new=True)
    current_tape.read_data(drive, TAPE_MOUNT)
elif action == 'list':
    files = {}
    for _, tape in tapes.items():
        for name, info in tape.files.items():
            if name in files and not info.is_better_than(files[name][1]):
                continue
            files[name] = (info, tape)

    for name, info_tuple in files.items():
        info, tape = info_tuple
        print('[%s] Name "%s", size %s, mtime %s' % (tape.label, name, format_size(info.size), format_mtime(info.mtime)))
elif action == 'find':
    best_info = None
    best_tape = None
    for _, tape in tapes.items():
        for name, info in tape.files.items():
            if name != args.files[1]:
                continue
            print('Found copy of file on "%s", size %s, mtime %s' % (tape.label, format_size(info.size), format_mtime(info.mtime)))
            if best_info is not None and best_info.is_better_than(info):
                continue
            best_info = info
            best_tape = tape
    if best_tape is not None:
        print('Best copy of file seems to be on "%s", size %s, mtime %s' % (best_tape.label, format_size(info.size), format_mtime(info.mtime)))
    else:
        print('Could not find that file :(')
elif action == 'mount':
    current_tape = get_current_tape(create_new=True)
    if current_tape is not None:
        drive.mount(TAPE_MOUNT)
        print('Mounted "%s" to "%s", run "umount %s" and wait for eject once done!' % (current_tape.label, TAPE_MOUNT, TAPE_MOUNT))
    else:
        print('Do not recognize this tape!')
        changer.eject()
elif action == 'statistics':
    for label, tape in tapes.items():
        print('[%s] Free = %s / %s (%.2f%%), Files = %d' % (label, format_size(tape.free), format_size(tape.size), (tape.free / tape.size) * 100.0, len(tape.files)))
