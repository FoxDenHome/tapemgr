from tape import Tape
from drive import Drive
from os import path, lstat, scandir
from subprocess import call
from pickle import load, dump
from sys import argv
from stat import S_ISDIR, S_ISREG

TAPE_MOUNT = '/mnt/tape'

drive = Drive('nst0')
tapes = {}
current_tape = None

TAPE_SIZE_SPARE = 1024 * 1024 # 1 MB

def save_tapes():
    fh = open('tapes.dat', 'wb')
    dump(tapes, fh)
    fh.close()

def load_tapes():
    global tapes
    try:
        fh = open('tapes.dat', 'rb')
        tapes = load(fh)
        fh.close()
    except:
        pass

def make_tape_label():
    idx = 0
    while True:
        idx += 1
        label = 'FoxDen %03d' % idx
        if label not in tapes:
            return label

def ask_for_tape(label):
    global current_tape

    if label is None:
        drive.eject()
        input('Please insert new/unused/blank tape and press return!')
        return format_current_tape()

    while current_tape is None or current_tape.label != label:
        drive.eject()
        input('Please insert tape "%s" and press return!' % label)
        current_tape = get_current_tape()

def get_current_tape():
    try:
        drive.read_label()
    except:
        drive.load()
    label = drive.read_label()
    if not label in tapes:
        return None
    return tapes[label]

def format_current_tape():
    if get_current_tape():
        raise ValueError('Tape is already in this program!')
    label = make_tape_label()
    drive.format(label)

    tape = Tape(label)
    tape.verify_in_drive(drive)
    tape.read_data(drive, TAPE_MOUNT)
    tapes[label] = tape
    save_tapes()
    return tape

def backup_file(file):
    global current_tape

    name = path.abspath(file.path)
    dir = path.dirname(name)

    fstat = file.stat(follow_symlinks=False)

    for _, tape in tapes.items():
        if name in tape.files and tape.files[name] >= fstat.st_mtime:
            print('[SKIP] %s' % name)
            return

    min_size = fstat.st_size + TAPE_SIZE_SPARE
    if current_tape is None or current_tape.free < min_size:
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
        current_tape.read_data(drive, TAPE_MOUNT, False)

    tape_name = '%s%s' % (TAPE_MOUNT, name)

    print('[STOR] %s' % name)
    call(['mkdir', '-p', '%s%s' % (TAPE_MOUNT, dir)])
    call(['cp', '-p', name, tape_name])
    current_tape.files[name] = fstat.st_mtime
    fstat_tape = lstat(tape_name)
    current_tape.free -= fstat_tape.st_size

def backup_recursive(dir):
    for file in scandir(dir):
            stat = file.stat(follow_symlinks=False)
            if S_ISDIR(stat.st_mode):
                backup_recursive(file.path)
            elif S_ISREG(stat.st_mode):
                backup_file(file)

load_tapes()

def format_size(size):
    for unit in ['','Ki','Mi','Gi','Ti','Pi','Ei','Zi']:
        if size < 1024.0:
            return '%3.1f %sB' % (size, unit)
        size /= 1024.0
    return '%.1f %sB' % (size, 'Yi')

if argv[1] == 'format':
    format_current_tape()
elif argv[1] == 'store':
    try:
        name = argv[2]
        stat = lstat(name)
        if S_ISDIR(stat.st_mode):
            backup_recursive(name)
        elif S_ISREG(stat.st_mode):
            backup_file(name)
        else:
            raise ValueError("Cannot backup file (not regular file or directory): %s" % name)
    finally:
        drive.unmount()
        save_tapes()
elif argv[1] == 'index':
    current_tape = get_current_tape()
    current_tape.read_data(drive, TAPE_MOUNT)
    save_tapes()
elif argv[1] == 'list':
    files = {}
    for _, tape in tapes.items():
        for name, mtime in tape.files.items():
            if name in files and files[name][1] >= mtime:
                continue
            files[name] = (mtime, tape)

    for name, info in files.items():
        mtime, tape = info
        print("%s [%s]" % (name, tape.label))
elif argv[1] == 'statistics':
    for label, tape in tapes.items():
        print('[%s] Free = %s / %s (%.2f%%), Files = %d' % (label, format_size(tape.free), format_size(tape.size), (tape.free / tape.size) * 100.0, len(tape.files)))
