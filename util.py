from subprocess import call, check_call, check_output

def logged_check_call(args):
    print('Running check_call', args)
    return check_call(args)

def logged_call(args):
    print('Running call', args)
    return call(args)

def logged_check_output(args):
    print('Running check_output', args)
    return check_output(args, encoding='utf-8')
