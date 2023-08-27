# pyright: reportImportCycles=false
from os import path, scandir
from pickle import load, dump
from typing import TYPE_CHECKING, Any, cast

if TYPE_CHECKING:
  from tape import Tape
else:
    Tape = Any

class Storage:
    dir: str
    tapes: dict[str, Tape]
    
    def __init__(self, dir: str) -> None:
        super().__init__()
        self.dir = dir
        self.tapes = {}
        self.load_all()

    def save(self, tape: Tape) -> None:
        fh = open(path.join(self.dir, tape.barcode), 'wb')
        dump(tape, fh)
        fh.close()
        self.tapes[tape.barcode] = tape

    def save_all(self) -> None:
        for tape in self.tapes.values():
            self.save(tape)

    def load_all(self) -> None:
        tapes: dict[str, Tape] = {}
        for file in scandir(self.dir):
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
        self.tapes = tapes
