#!/usr/bin/env python3

import os
import sys
import time
import json
import socket
import signal
import libtorrent as lt
from threading import Thread
from daemonize import Daemonize
from pathlib import Path

PID_FILE = "/tmp/torrcli_daemon.pid"
SOCKET_PATH = "/tmp/torrcli_daemon.sock"
DOWNLOAD_DIR = Path.home() / 'Downloads'

if not os.path.exists(DOWNLOAD_DIR):
    os.makedirs(DOWNLOAD_DIR)

def download_magnet(magnet_link, save_path):
    ses = lt.session()
    ses.listen_on(6881, 6891)
    params = {'save_path': save_path, 'storage_mode': lt.storage_mode_t(2)}
    handle = lt.add_magnet_uri(ses, magnet_link, params)

    while not handle.has_metadata():
        time.sleep(1)

    while not handle.is_seed():
        time.sleep(1) 

def worker_thread(magnet_link, save_path):
    try:
        print(f"Starting download: {magnet_link} -> {save_path}")
        download_magnet(magnet_link, save_path)
        print(f"Completed: {magnet_link}")
    except Exception as e:
        print(f"Error downloading {magnet_link}: {e}")

def socket_server():
    if os.path.exists(SOCKET_PATH):
        os.remove(SOCKET_PATH)

    with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as server:
        server.bind(SOCKET_PATH)
        server.listen(5)

        while True:
            conn, _ = server.accept()
            with conn:
                try:
                    raw = conn.recv(4096)
                    data = json.loads(raw.decode("utf-8"))
                    magnet_link = data.get("magnet_link")
                    save_path = data.get("save_path") or DOWNLOAD_DIR 
                    conn.sendall(f"error: {str(save_path)}\n".encode())

                    if not os.path.exists(save_path):
                        os.makedirs(save_path)

                    if magnet_link:
                        t = Thread(target=worker_thread, args=(magnet_link, save_path), daemon=True)
                        t.start()
                        conn.sendall(b"started\n")
                    else:
                        conn.sendall(b"no magnet_link provided\n")
                except Exception as e:
                    conn.sendall(f"error: {str(e)}\n".encode())

def clean_exit(*args):
    if os.path.exists(SOCKET_PATH):
        os.remove(SOCKET_PATH)
    sys.exit(0)

def main():
    signal.signal(signal.SIGTERM, clean_exit)
    signal.signal(signal.SIGINT, clean_exit)
    socket_server()

if __name__ == "__main__":
    daemon = Daemonize(app="torrent_daemon", pid=PID_FILE, action=main)
    daemon.start()
