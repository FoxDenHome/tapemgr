# pyright: reportImportCycles=false
from os import path, scandir
from json import dump, load
from util import is_dry_run
from traceback import print_exc
from tape import Tape

class Storage:
    dir: str
    tapes: dict[str, Tape]

    def __init__(self, dir: str) -> None:
        super().__init__()
        self.dir = dir
        self.tapes = {}
        self.load_all()

    def save(self, tape: Tape) -> None:
        if is_dry_run():
            return

        fh = open(path.join(self.dir, f"{tape.barcode}.json"), 'w')
        dump(
            tape.as_dict(),
            fh,
            indent=4,
            sort_keys=True,
        )
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
                fh = open(file.path, 'r')
                tape = Tape.from_dict(load(fh))
                tapes[tape.barcode] = tape
                fh.close()
            except:
                print('Error loading tape data "%s"' % file.name)
                print_exc()
        self.tapes = tapes
