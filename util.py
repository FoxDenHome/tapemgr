from subprocess import call, check_call, check_output
from datetime import datetime
from os import readlink
from os.path import join as path_join, dirname

def logged_check_call(args: list[str]):
    print('Running check_call', args)
    _ = check_call(args)

def logged_call(args: list[str]):
    print('Running call', args)
    _ = call(args)

def logged_check_output(args: list[str]) -> str:
    print('Running check_output text', args)
    return check_output(args, encoding='utf-8')

def logged_check_output_binary(args: list[str]) -> bytes:
    print('Running check_output binary', args)
    return check_output(args)

def format_size(size: float):
    for unit in ['','Ki','Mi','Gi','Ti','Pi','Ei','Zi']:
        if size < 1024.0:
            return '%3.1f %sB' % (size, unit)
        size /= 1024.0
    return '%.1f %sB' % (size, 'Yi')

def format_mtime(mtime: float):
    time = datetime.fromtimestamp(mtime)
    return time.strftime('%Y-%m-%d %H:%M:%S')

def resolve_symlink(path: str) -> str:
   try:
        resolved = readlink(path)
        return resolve_symlink(path_join(dirname(path), resolved))
   except OSError: # if the last is not symbolic file will throw OSError
        return path
