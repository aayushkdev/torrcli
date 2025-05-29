import sys
import tty
import termios
import selectors

def format_size(size_in_bytes):
    for unit in ["B", "KB", "MB", "GB", "TB"]:
        if size_in_bytes < 1024:
            return f"{size_in_bytes:.2f} {unit}"
        size_in_bytes /= 1024
    return f"{size_in_bytes:.2f} PB"

def format_speed(speed_kbps):
    return format_size(speed_kbps) + "/s"

def format_time(seconds):
    seconds = int(seconds)
    if seconds < 60:
        return f"{seconds} sec"
    elif seconds < 3600:
        mins, sec = divmod(seconds, 60)
        return f"{mins} min {sec} sec"
    elif seconds < 86400:
        hours, rem = divmod(seconds, 3600)
        mins, sec = divmod(rem, 60)
        return f"{hours} hr {mins} min"
    else:
        days, rem = divmod(seconds, 86400)
        hours, rem = divmod(rem, 3600)
        mins, sec = divmod(rem, 60)
        return f"{days} d {hours} hr {mins} min"

def setup_nonblocking_input():
    fd = sys.stdin.fileno()
    tty.setcbreak(fd) 
    import os
    os.set_blocking(fd, False)

def restore_input_mode():
    fd = sys.stdin.fileno()
    termios.tcsetattr(fd, termios.TCSADRAIN, termios.tcgetattr(fd))

def get_pressed_key():
    sel = selectors.DefaultSelector()
    sel.register(sys.stdin, selectors.EVENT_READ)
    events = sel.select(timeout=0)
    if events:
        return sys.stdin.read(1)
    return None
