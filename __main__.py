from tape import FileInfo, Tape
from drive import Drive
from os import path, lstat, scandir
from subprocess import call
from pickle import load, dump
from sys import argv
from stat import S_ISDIR, S_ISREG

TAPE_MOUNT = '/mnt/tape'
TAPES_DAT_FILE = path.join(path.dirname(__file__), 'tapes.dat')

drive = Drive('nst0')
tapes = {}
current_tape = None

TAPE_SIZE_SPARE = 1024 * 1024 * 1024 # 1 GB

def save_tapes():
    fh = open(TAPES_DAT_FILE, 'wb')
    dump(tapes, fh)
    fh.close()

def load_tapes():
    global tapes
    try:
        fh = open(TAPES_DAT_FILE, 'rb')
        tapes = load(fh)
        fh.close()
    except:
        pass

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
        input('Please insert new/unused/blank tape and press return!')
        format_current_tape()
        current_tape = get_current_tape()
        return

    while True:
        current_tape = get_current_tape()
        if current_tape and current_tape.label == label:
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
    print ('Formatted tape with label "%s"!' % label)
    return tape

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
        current_tape.read_data(drive, TAPE_MOUNT, False)
        save_tapes()

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
        current_tape.read_data(drive, TAPE_MOUNT, False)
        save_tapes()

    tape_name = '%s%s' % (TAPE_MOUNT, name)

    call(['mkdir', '-p', '%s%s' % (TAPE_MOUNT, dir)])
    call(['cp', name, tape_name])
    
    fstat_tape = lstat(tape_name)
    current_tape.files[name] = FileInfo(size=fstat_tape.st_size,mtime=fstat_tape.st_mtime)

    current_tape.read_data(drive, TAPE_MOUNT, False)
    save_tapes()

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
        save_tapes()
elif argv[1] == 'index':
    current_tape = get_current_tape(create_new=True)
    current_tape.read_data(drive, TAPE_MOUNT)
    save_tapes()
elif argv[1] == 'list':
    files = {}
    for _, tape in tapes.items():
        for name, info in tape.files.items():
            if name in files and not info.is_better_than(files[name][1]):
                continue
            files[name] = (info, tape)

    for name, info_tuple in files.items():
        info, tape = info_tuple
        print('%s [%s]' % (name, tape.label))
elif argv[1] == 'mount':
    current_tape = get_current_tape(create_new=True)
    if current_tape is not None:
        drive.mount(TAPE_MOUNT)
        print('Once done, run "umount %s"!' % TAPE_MOUNT)
    else:
        print('Do not recognize this tape!')
elif argv[1] == 'statistics':
    for label, tape in tapes.items():
        print('[%s] Free = %s / %s (%.2f%%), Files = %d' % (label, format_size(tape.free), format_size(tape.size), (tape.free / tape.size) * 100.0, len(tape.files)))
