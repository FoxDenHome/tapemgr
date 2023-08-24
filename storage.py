# pyright: reportImportCycles=false
from os import path, scandir
from pickle import load, dump
from typing import TYPE_CHECKING, Any, cast

if TYPE_CHECKING:
  from tape import Tape
else:
    Tape = Any

tapes_dir: str | None = None

def set_storage_dir(dir: str):
    global tapes_dir
    tapes_dir = dir

def save_tape(tape: Tape):
    if not tapes_dir:
        raise Exception("Tapes dir not set")
    fh = open(path.join(tapes_dir, tape.barcode), 'wb')
    dump(tape, fh)
    fh.close()

def load_all_tapes():
    tapes: dict[str, Tape] = {}
    for file in scandir(tapes_dir):
        if not file.is_file():
            continue
        if file.name[0] == '.':
            continue
        try:
            fh = open(file.path, 'rb')
            tape = cast(Tape, load(fh))
            tapes[tape.barcode] = tape
            fh.close()
        except:
            print('Error loading tape data "%s"' % file.name)
    return tapes
