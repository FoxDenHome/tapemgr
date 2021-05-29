from . import Drive, Tape
from os import path, lstat
from subprocess import call

TAPE_MOUNT = '/mnt/tape'

drive = Drive('nst0')
tapes = {}
files = set()

def save_tapes():
    pass

def load_tapes():
    pass

def recompute_files():
    files.clear()
    for tape in tapes:
        for file in tape.files:
            files.add(file)

def make_tape_label():
    idx = 0
    while True:
        idx += 1
        label = "FoxDen %03d" % idx
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
        raise ValueError("Tape is already in this program!")
    label = make_tape_label()
    drive.format(label)

    tape = Tape(label)
    tape.verify_in_drive(drive)
    tape.read_data(drive)
    tapes[label] = tape
    save_tapes()

def backup_file(name):
    name = path.abspath(name)
    dir = path.basename(name)

    fstat = lstat(name)
    fstat.st_size

    call(['mkdir', '-p', '%s/%s' % (TAPE_MOUNT, dir)])
