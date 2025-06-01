import configparser
from pathlib import Path
import shutil

DEFAULT_CONF_PATH = Path.home() / ".config" / "torrcli" / "torrcli.conf"
DEFAULT_CONF_TEMPLATE = Path("/usr/share/torrcli/torrcli.conf.example")

DEFAULT_CONF_PATH.parent.mkdir(parents=True, exist_ok=True)
if not DEFAULT_CONF_PATH.exists() and DEFAULT_CONF_TEMPLATE.exists():
    shutil.copy(DEFAULT_CONF_TEMPLATE, DEFAULT_CONF_PATH)

config = configparser.ConfigParser()
config.read(DEFAULT_CONF_PATH)

DEFAULTS = {
    "general": {
        "socket_path": "/tmp/torrcli_daemon.sock",
        "pid_file": "/tmp/torrcli_daemon.pid",
        "data_dir": str(Path.home() / ".local/share/torrcli"),
        "default_save_path": str(Path.home() / "Downloads"),
        "log_level": "INFO",
    },
    "network": {
        "listen_interfaces": "0.0.0.0:6881",
        "dht_enabled": True,
        "lsd_enabled": False,
        "upnp_enabled": True,
        "natpmp_enabled": True,
        "outgoing_utp_enabled": True,
        "incoming_utp_enabled": True,
        "outgoing_tcp_enabled": True,
        "incoming_tcp_enabled": True,
    },
    "limits": {
        "max_download_speed": 0,
        "max_upload_speed": 0,
        "max_active_downloads": 3,
        "max_active_seeds": 5,
        "share_ratio_limit": 200,
    },
    "resume": {
        "auto_start": True,
        "save_resume_data_interval": 300,
    },
    "security": {
        "anonymous_mode": False,
        "validate_https_trackers": True,
        "dht_privacy_lookups": True,
        "dht_ignore_dark_internet": True,
        "ssrf_mitigation": True,
        "out_enc_policy": 1,
        "in_enc_policy": 1,
        "allowed_enc_level": 3,
        "prefer_rc4": False,
    },
    "proxy": {
        "proxy_type": 0,
        "proxy_hostname": "127.0.0.1",
        "proxy_port": 9050,
        "proxy_username": "",
        "proxy_password": "",
        "proxy_hostnames": True,
        "proxy_peer_connections": True,
        "proxy_tracker_connections": True,
        "force_proxy": True,
    }
}

def get(key, section="general", fallback=None, cast=str):
    def cast_value(val):
        if cast is bool:
            val_str = str(val).strip().lower()
            if val_str == "true":
                return True
            elif val_str == "false":
                return False
            raise ValueError(f"Cannot cast {val!r} to bool")
        return cast(val)

    if config.has_option(section, key):
        try:
            return cast_value(config.get(section, key))
        except Exception:
            pass

    if section in DEFAULTS and key in DEFAULTS[section]:
        try:
            return cast_value(DEFAULTS[section][key])
        except Exception:
            pass

    return fallback

SOCKET_PATH = get("socket_path")
PID_FILE = get("pid_file")
DATA_DIR = Path(get("data_dir")).expanduser()
DEFAULT_SAVE_PATH = Path(get("default_save_path")).expanduser()
LOG_LEVEL = get("log_level")

LISTEN_INTERFACES = get("listen_interfaces", "network")
DHT_ENABLED = get("dht_enabled", "network", cast=bool)
LSD_ENABLED = get("lsd_enabled", "network", cast=bool)
UPNP_ENABLED = get("upnp_enabled", "network", cast=bool)
NATPMP_ENABLED = get("natpmp_enabled", "network", cast=bool)
OUTGOING_UTP_ENABLED = get("outgoing_utp_enabled", "network", cast=bool)
INCOMING_UTP_ENABLED = get("incoming_utp_enabled", "network", cast=bool)
OUTGOING_TCP_ENABLED = get("outgoing_tcp_enabled", "network", cast=bool)
INCOMING_TCP_ENABLED = get("incoming_tcp_enabled", "network", cast=bool)

MAX_DOWNLOAD_SPEED = get("max_download_speed", "limits", cast=int)
MAX_UPLOAD_SPEED = get("max_upload_speed", "limits", cast=int)
MAX_ACTIVE_DOWNLOADS = get("max_active_downloads", "limits", cast=int)
MAX_ACTIVE_SEEDS = get("max_active_seeds", "limits", cast=int)
SHARE_RATIO_LIMIT = get("share_ratio_limit", "limits", cast=int)

AUTO_START = get("auto_start", "resume", cast=bool)
SAVE_RESUME_DATA_INTERVAL = get("save_resume_data_interval", "resume", cast=int)

ANONYMOUS_MODE = get("anonymous_mode", "security", cast=bool)
VALIDATE_HTTPS_TRACKERS = get("validate_https_trackers", "security", cast=bool)
DHT_PRIVACY_LOOKUPS = get("dht_privacy_lookups", "security", cast=bool)
DHT_IGNORE_DARK_INTERNET = get("dht_ignore_dark_internet", "security", cast=bool)
SSRF_MITIGATION = get("ssrf_mitigation", "security", cast=bool)
OUT_ENC_POLICY = get("out_enc_policy", "security", cast=int)
IN_ENC_POLICY = get("in_enc_policy", "security", cast=int)
ALLOWED_ENC_LEVEL = get("allowed_enc_level", "security", cast=int)
PREFER_RC4 = get("prefer_rc4", "security", cast=bool)

PROXY_TYPE = get("proxy_type", "proxy", cast=int)
PROXY_HOSTNAME = get("proxy_hostname", "proxy")
PROXY_PORT = get("proxy_port", "proxy", cast=int)
PROXY_USERNAME = get("proxy_username", "proxy")
PROXY_PASSWORD = get("proxy_password", "proxy")
PROXY_HOSTNAMES = get("proxy_hostnames", "proxy", cast=bool)
PROXY_PEER_CONNECTIONS = get("proxy_peer_connections", "proxy", cast=bool)
PROXY_TRACKER_CONNECTIONS = get("proxy_tracker_connections", "proxy", cast=bool)
FORCE_PROXY = get("force_proxy", "proxy", cast=bool)

DATA_DIR.mkdir(parents=True, exist_ok=True)