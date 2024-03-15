from dataclasses import dataclass
from types import FrameType
from typing import cast
from drive import Drive
from changer import Changer
from os import path, lstat
from stat import S_ISDIR, S_ISREG
from storage import Storage
from argparse import ArgumentParser
from signal import SIGINT, SIGTERM, signal
from util import format_size, format_mtime, logged_check_call, logged_call
from manager import Manager
from os.path import dirname, exists

@dataclass
class ArgParseResult:
    mount: str
    tape_prefix: str
    tape_type: str
    tape_dir: str
    tape_suffix: str
    device: str
    age_recipients: str
    filename_key: str
    changer: str
    changer_drive_index: int
    include_hidden: bool
    action: list[str]
    files: list[str]

parser = ArgumentParser(description='Tape manager')
_ = parser.add_argument('action', metavar='action', type=str, nargs=1, help='The action to perform')
_ = parser.add_argument('files', metavar='files', type=str, nargs='*', help='Files to store (for store action)')
_ = parser.add_argument('--device', dest='device', type=str, default='AUTO')
_ = parser.add_argument('--changer', dest='changer', type=str, default='/dev/sch0')
_ = parser.add_argument('--changer-drive-index', dest='changer_drive_index', type=int, default=0)
_ = parser.add_argument('--mount', dest='mount', type=str, default='/mnt/tape')
_ = parser.add_argument('--tape-dir', dest='tape_dir', type=str, default=path.join(path.dirname(__file__), 'tapes'))
_ = parser.add_argument('--tape-prefix', dest='tape_prefix', type=str, default='P', help='Prefix to add to tape label and barcode')
_ = parser.add_argument('--tape-suffix', dest='tape_suffix', type=str, default='S', help='Suffix to add to tape label and barcode')
_ = parser.add_argument('--tape-type', dest='tape_type', type=str, default='L6', help='Tape type (L6 for LTO-6, L7 for LTO-7 etc)')
_ = parser.add_argument('--age-recipients', dest='age_recipients', type=str, default='/mnt/keydisk/tape-age.pub', help='Age recipients file for encryption')
_ = parser.add_argument('--filename-key', dest='filename_key', type=str, default='/mnt/keydisk/tape-filename.key', help='Key for filename encryption')
_ = parser.add_argument('--include-hidden', dest='include_hidden', action='store_true', help='Include hidden files in backup (default: false)')

args = cast(ArgParseResult, parser.parse_args())

NO_TAPE_ACTIONS = {'list', 'statistics', 'find'}

if len(args.tape_type) != 2 or args.tape_type[0] != 'L':
    raise ValueError('Tape type must be L#')

action = args.action[0]

# We don't need to poke the changer for actions that don't involve actually talking to the tape drive
if args.device == 'AUTO' and action not in NO_TAPE_ACTIONS:
    from scsi import find_dte_path_by_index
    args.device = find_dte_path_by_index(changer_device=args.changer, index=args.changer_drive_index)
    print(f'Successfully found tape drive node {args.device}')

manager = Manager(Drive(args.device), Changer(args.changer, args.changer_drive_index), Storage(args.tape_dir), args.age_recipients, args.filename_key)
manager.set_barcode(args.tape_prefix, args.tape_suffix, args.tape_type)
manager.mountpoint = args.mount
manager.include_hidden = args.include_hidden

def signal_exit_handler(_sig: int, _frame: FrameType | None) -> None:
    global should_exit
    should_exit = True
    print('Got exit signal, exiting ASAP...')
_ = signal(SIGINT, signal_exit_handler)
_ = signal(SIGTERM, signal_exit_handler)


if action == 'format':
    manager.format_current_tape()
elif action == 'store':
    try:
        for name in args.files:
            stat = lstat(name)
            if S_ISDIR(stat.st_mode):
                manager.backup_recursive(name)
            elif S_ISREG(stat.st_mode):
                manager.backup_file(name, stat)
            else:
                raise ValueError('Cannot backup file (not regular file or directory): %s' % name)
        print('Checking for files to tombstone...')
        manager.backup_tombstone()
    finally:
        manager.shutdown()
elif action == 'unload':
    manager.shutdown()
elif action == 'index':
    try:
        manager.index_tape(args.files[0])
    finally:
        manager.shutdown()
elif action == 'list':
    files = manager.list_all_best()
    for encrypted_name, info_tuple in files.items():
        info, tape = info_tuple
        name = manager.decrypt_filename(encrypted_name)
        print('[%s] Name "%s", size %s, mtime %s' % (tape.barcode, name, format_size(info.size), format_mtime(info.mtime)))
elif action == 'find':
    best_info, best_tape = manager.find(args.files[0])
    print('Best copy of file seems to be on "%s", size %s, mtime %s' % (best_tape.barcode, format_size(best_info.size), format_mtime(best_info.mtime)))
elif action == 'mount':
    manager.mount(args.files[0])
elif action == 'copyback':
    try:
        tape_barcode = args.files[0]
        decryption_key = args.files[1]
        manager.mount(tape_barcode)
        dst = args.files[2]
        to_copy = set(args.files[3:])
        all_files = manager.list_all_best()

        for encrypted_name, info_tuple in all_files.items():
            info, tape = info_tuple
            if tape.barcode != tape_barcode:
                continue
            
            name = manager.decrypt_filename(encrypted_name)
            if (name not in to_copy) and ("*" not in to_copy):
                continue

            dst_name = path.join(dst, name)
            src_name = path.join(manager.mountpoint, encrypted_name)
            if exists(dst_name):
                print('Skipping "%s" -> "%s" (already exists)' % (src_name, dst_name))
                continue
            print('Copying "%s" -> "%s"' % (src_name, dst_name))
            logged_check_call(['mkdir', '-p', dirname(dst_name)])
            logged_call(['age', '-d', '-o', dst_name, '-i', decryption_key, src_name])
            logged_call(['touch', '-r', src_name, dst_name])
    finally:
        manager.shutdown()
elif action == 'statistics':
    for _, tape in sorted(manager.storage.tapes.items()):
        print('[%s] Free = %s / %s (%.2f%%), Files = %d' % (tape.barcode, format_size(tape.free), format_size(tape.size), (tape.free / tape.size) * 100.0, len(tape.files)))
elif action == 'export':
    manager.changer.export_to_iobay_by_barcode(args.files[0])
elif action == 'import':
    manager.changer.import_from_iobay()
