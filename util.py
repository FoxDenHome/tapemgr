from subprocess import call, check_call, check_output
from datetime import datetime

def logged_check_call(args):
    print('Running check_call', args)
    return check_call(args)

def logged_call(args):
    print('Running call', args)
    return call(args)

def logged_check_output(args):
    print('Running check_output', args)
    return check_output(args, encoding='utf-8')

def format_size(size):
    for unit in ['','Ki','Mi','Gi','Ti','Pi','Ei','Zi']:
        if size < 1024.0:
            return '%3.1f %sB' % (size, unit)
        size /= 1024.0
    return '%.1f %sB' % (size, 'Yi')

def format_mtime(mtime):
    time = datetime.fromtimestamp(mtime)
    return time.strftime("%Y-%m-%d %H:%M:%S")
