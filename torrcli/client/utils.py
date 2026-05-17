import sys
import tty
import termios
import selectors

_orig_termios = None
_selector = None

def format_size(size_in_bytes):
    for unit in ["B", "KB", "MB", "GB", "TB"]:
        if size_in_bytes < 1024:
            return f"{size_in_bytes:.2f} {unit}"
        size_in_bytes /= 1024
    return f"{size_in_bytes:.2f} PB"

def format_speed(speed_bps):
    return format_size(speed_bps) + "/s"

def format_time(seconds):
    if seconds < 0:
        return "\u221e sec"
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
    global _orig_termios, _selector
    if not sys.stdin.isatty():
        return
    if _selector is not None:
        try:
            _selector.close()
        except Exception:
            pass
    if _orig_termios is not None:
        _orig_termios = None
    fd = sys.stdin.fileno()
    _orig_termios = termios.tcgetattr(fd)
    tty.setcbreak(fd)
    import os
    os.set_blocking(fd, False)
    _selector = selectors.DefaultSelector()
    _selector.register(sys.stdin, selectors.EVENT_READ)

def restore_input_mode():
    global _orig_termios, _selector
    if _orig_termios is not None:
        try:
            fd = sys.stdin.fileno()
            termios.tcsetattr(fd, termios.TCSADRAIN, _orig_termios)
        except Exception:
            pass
        _orig_termios = None
    if _selector is not None:
        try:
            _selector.close()
        except Exception:
            pass
        _selector = None

def get_pressed_key():
    if _selector is None:
        return None
    events = _selector.select(timeout=0)
    if events:
        return sys.stdin.read(1)
    return None
