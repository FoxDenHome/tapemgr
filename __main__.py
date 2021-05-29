from tape import Tape
from drive import Drive
from os import path, lstat
from subprocess import call
from pickle import load, dump
from sys import argv

TAPE_MOUNT = '/mnt/tape'

drive = Drive('nst0')
tapes = {}

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

def backup_file(name):
    name = path.abspath(name)
    dir = path.dirname(name)

    fstat = lstat(name)

    for _, tape in tapes.items():
        if name in tape.files and tape.files[name] >= fstat.st_mtime:
            return

    current_tape = get_current_tape()
    if fstat.st_size >= current_tape.free - TAPE_SIZE_SPARE:
        # Find new tape!
        return

    call(['mkdir', '-p', '%s%s' % (TAPE_MOUNT, dir)])
    call(['cp', '-p', name, '%s%s' % (TAPE_MOUNT, name)])
    current_tape.files[name] = fstat.st_mtime
    save_tapes()

load_tapes()

if argv[1] == '--format':
    format_current_tape()
elif argv[1] == '--store':
    drive.mount(TAPE_MOUNT)
    backup_file(argv[2])
    drive.unmount()
elif argv[1] == '--list':
    files = {}
    for _, tape in tapes.items():
        for name, mtime in tape.files.items():
            if name in files and files[name][1] >= mtime:
                continue
            files[name] = (mtime, tape)

    for name, info in files.items():
        print("%s [%s]" % (name, info.tape.label))
