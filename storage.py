from os import path, scandir
from pickle import load, dump

TAPES_DIR = path.join(path.dirname(__file__), 'tapes')

def save_tape(tape):
    fh = open(path.join(TAPES_DIR, tape.label), 'wb')
    dump(tape, fh)
    fh.close()

def load_all_tapes():
    tapes = {}
    for file in scandir(TAPES_DIR):
        if not file.is_file():
            continue
        if file.name[0] == '.':
            continue
        try:
            fh = open(file.path, 'rb')
            tape = load(fh)
            tapes[tape.label] = tape
            fh.close()
        except:
            print('Error loading tape data "%s"' % file.name)
    return tapes
