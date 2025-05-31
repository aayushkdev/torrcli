from pathlib import Path
from appdirs import user_data_dir

PID_FILE = "/tmp/torrcli_daemon.pid"
SOCKET_PATH = "/tmp/torrcli_daemon.sock"

DATA_DIR = Path(user_data_dir("torrcli"))
TORRENT_DATA_DIR = DATA_DIR / "downloads_data"
TORRENT_DATA_DIR.mkdir(parents=True, exist_ok=True)

DEFAULT_SAVE_PATH = str(Path.home() / "Downloads")