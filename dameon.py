import sys
import json
import time
import signal
import socket
from pathlib import Path
import libtorrent as lt

PID_FILE = "/tmp/torrcli_daemon.pid"
SOCKET_PATH = "/tmp/torrcli_daemon.sock"

ses = lt.session({'listen_interfaces': '0.0.0.0:6881'})

torrent_handles = {}

def clean_exit(*_):
    """Clean up when exiting."""
    Path(SOCKET_PATH).unlink(missing_ok=True)
    sys.exit(0)

def handle_request(data, conn):
    """Handle incoming request and perform actions."""
    req_type = data.get("type")
    source = data.get("source")
    save_path = data.get("save_path")

    if req_type == "add_torrent":
        if source.startswith("magnet:"):
            params = {"save_path": save_path, "storage_mode": lt.storage_mode_t(2)}
            handle = ses.add_torrent({"url":source, "save_path":save_path})
            torrent_handles[source] = handle

            # Wait indefinitely for metadata to be available
            while not handle.has_metadata():
                time.sleep(1)
            handle.pause()
            ti = handle.get_torrent_info()
        else:
            info = lt.torrent_info(source)
            params = {"save_path": save_path, "ti": info}
            handle = ses.add_torrent(params)
            handle.pause()
            torrent_handles[source] = handle
            ti = info

        status = handle.status()
        metadata = {
            "name": ti.name(),
            "size_mb": f"{ti.total_size() / (1024 ** 2):.2f} MB",
            "num_files": ti.num_files(),
            "seeders": status.num_seeds,
            "leechers": status.num_peers - status.num_seeds,
        }
        conn.sendall(json.dumps({"status": "metadata", "data": metadata}).encode())

    elif req_type == "start_download":
        handle = torrent_handles.get(source)
        if handle:
            handle.resume()
            conn.sendall(b'{"status": "resumed"}')
        else:
            conn.sendall(b'{"status": "error", "message": "Torrent not found"}')

    elif req_type == "pause_download":
        handle = torrent_handles.get(source)
        if handle:
            handle.pause()
            conn.sendall(b'{"status": "paused"}')
        else:
            conn.sendall(b'{"status": "error", "message": "Torrent not found"}')

    elif req_type == "remove_download":
        handle = torrent_handles.pop(source, None)
        if handle:
            ses.remove_torrent(handle)
            conn.sendall(b'{"status": "removed"}')
        else:
            conn.sendall(b'{"status": "error", "message": "Torrent not found"}')

    else:
        conn.sendall(b'{"status": "error", "message": "Unknown request type"}')

def socket_server():
    """Main server loop for receiving requests."""
    Path(SOCKET_PATH).unlink(missing_ok=True)

    with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as server:
        server.bind(SOCKET_PATH)
        server.listen(5)

        while True:
            conn, _ = server.accept()
            with conn:
                raw = conn.recv(4096)
                if not raw:
                    continue

                data = json.loads(raw.decode("utf-8"))
                handle_request(data, conn)

def main():
    signal.signal(signal.SIGTERM, clean_exit)
    signal.signal(signal.SIGINT, clean_exit)
    socket_server()

if __name__ == "__main__":
    main()