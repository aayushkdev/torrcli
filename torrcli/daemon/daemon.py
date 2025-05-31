import signal
import asyncio
from torrcli.daemon.session import load_resume_and_torrents
from torrcli.daemon.alerts import clean_exit
from torrcli.daemon.ipc_server import socket_server

def main():
    load_resume_and_torrents()
    signal.signal(signal.SIGTERM, clean_exit)
    signal.signal(signal.SIGINT, clean_exit)
    asyncio.run(socket_server())

if __name__ == "__main__":
    main()