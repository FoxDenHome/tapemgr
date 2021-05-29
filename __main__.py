from tape import FileInfo, Tape
from drive import Drive
from os import path, lstat, scandir
from subprocess import call
from sys import argv
from stat import S_ISDIR, S_ISREG
from storage import save_tape, load_all_tapes
from datetime import datetime

TAPE_MOUNT = '/mnt/tape'

drive = Drive('nst0')
tapes = {}
current_tape = None

TAPE_SIZE_SPARE = 1024 * 1024 * 1024 # 1 GB

def save_all_tapes():
    for _, tape in tapes.items():
        save_tape(tape)

def refresh_current_tape():
    current_tape.read_data(drive, TAPE_MOUNT, False)

def make_tape_label():
    idx = 0
    while True:
        idx += 1
        label = 'FOX%03d' % idx
        if label not in tapes:
            return label

def ask_for_tape(label):
    global current_tape

    if label is None:
        drive.eject()
        new_label = make_tape_label()
        input('Please insert new/unused/blank tape and press return! (label will be "%d")' % new_label)
        format_current_tape(new_label, True)
        return

    while True:
        current_tape = get_current_tape()
        if current_tape and current_tape.label == label:
            drive.mount()
            return
        drive.eject()
        input('Please insert tape "%s" and press return!' % label)

def get_current_tape(create_new=False):
    try:
        drive.read_label()
    except:
        drive.load()
    label = drive.read_label()
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
    drive.format(label)

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

    call(['mkdir', '-p', '%s%s' % (TAPE_MOUNT, dir)])
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

tapes = load_all_tapes()

def format_size(size):
    for unit in ['','Ki','Mi','Gi','Ti','Pi','Ei','Zi']:
        if size < 1024.0:
            return '%3.1f %sB' % (size, unit)
        size /= 1024.0
    return '%.1f %sB' % (size, 'Yi')

def format_mtime(mtime):
    time = datetime.fromtimestamp(mtime)
    return time.strftime("%Y-%m-%d %H:%M:%S")

if argv[1] == 'format':
    format_current_tape()
elif argv[1] == 'store':
    try:
        for name in argv[2:]:
            stat = lstat(name)
            if S_ISDIR(stat.st_mode):
                backup_recursive(name)
            elif S_ISREG(stat.st_mode):
                backup_file(name)
            else:
                raise ValueError('Cannot backup file (not regular file or directory): %s' % name)
    finally:
        drive.unmount()
elif argv[1] == 'index':
    current_tape = get_current_tape(create_new=True)
    current_tape.read_data(drive, TAPE_MOUNT)
elif argv[1] == 'list':
    files = {}
    for _, tape in tapes.items():
        for name, info in tape.files.items():
            if name in files and not info.is_better_than(files[name][1]):
                continue
            files[name] = (info, tape)

    for name, info_tuple in files.items():
        info, tape = info_tuple
        print('[%s] Name "%s", size %s, mtime %s' % (tape.label, name, format_size(info.size), format_mtime(info.mtime)))
elif argv[1] == 'find':
    best_info = None
    best_tape = None
    for _, tape in tapes.items():
        for name, info in tape.files.items():
            if name != argv[2]:
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
elif argv[1] == 'mount':
    current_tape = get_current_tape(create_new=True)
    if current_tape is not None:
        drive.mount(TAPE_MOUNT)
        print('Mounted "%s" to "%s", run "umount %s" and wait for eject once done!' % (current_tape.label, TAPE_MOUNT, TAPE_MOUNT))
    else:
        print('Do not recognize this tape!')
        drive.eject()
elif argv[1] == 'statistics':
    for label, tape in tapes.items():
        print('[%s] Free = %s / %s (%.2f%%), Files = %d' % (label, format_size(tape.free), format_size(tape.size), (tape.free / tape.size) * 100.0, len(tape.files)))
