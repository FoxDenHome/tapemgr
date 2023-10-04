from subprocess import call, check_call, check_output
from datetime import datetime

def logged_check_call(args: list[str]):
    print('Running check_call', args)
    _ = check_call(args)

def logged_call(args: list[str]):
    print('Running call', args)
    _ = call(args)

def logged_check_output(args: list[str], encoding: str | None = 'utf-8') -> str:
    print('Running check_output', args)
    return check_output(args, encoding=encoding)

def format_size(size: float):
    for unit in ['','Ki','Mi','Gi','Ti','Pi','Ei','Zi']:
        if size < 1024.0:
            return '%3.1f %sB' % (size, unit)
        size /= 1024.0
    return '%.1f %sB' % (size, 'Yi')

def format_mtime(mtime: float):
    time = datetime.fromtimestamp(mtime)
    return time.strftime('%Y-%m-%d %H:%M:%S')
