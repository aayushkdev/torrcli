import libtorrent as lt
import time
import json
import os
import socket
from rich.console import Console
from rich.progress import Progress
from rich.table import Table
from rich.prompt import Confirm

SOCKET_PATH = "/tmp/torrcli_daemon.sock"
console = Console()

def show_metadata(handle):
    s = handle.status()
    ti = handle.get_torrent_info()

    name = ti.name()
    size_mb = f"{ti.total_size() / (1024 ** 2):.2f} MB"
    num_files = ti.num_files()
    num_peers = s.num_peers
    seeders = s.num_seeds
    leechers = num_peers - seeders
    trackers = list(ti.trackers())

    table = Table(title="Torrent Metadata", show_lines=True)
    table.add_column("Name", style="bold cyan")
    table.add_column("Size", style="green")
    table.add_column("Files", style="green")
    table.add_column("Seeders", style="yellow")
    table.add_column("Leechers", style="yellow")
    table.add_column("Trackers", style="yellow")

    table.add_row(
        name,
        size_mb,
        str(num_files),
        str(seeders),
        str(leechers),
        str(len(trackers)),
    )

    console.print(table)


def send_to_daemon(data):
    if not os.path.exists(SOCKET_PATH):
        console.print("[red]Daemon is not running![/red]")
        return

    try:
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as sock:
            sock.connect(SOCKET_PATH)
            sock.sendall(json.dumps(data).encode())
            response = sock.recv(1024).decode()
            console.print(f"[green]Daemon response:[/green] {response}")
    except Exception as e:
        console.print(f"[red]Error communicating with daemon: {e}[/red]")


def download_from_magnet(magnet_link, save_path):
    ses = lt.session()
    ses.listen_on(6881, 6891)
    params = {
        'save_path': '/tmp',
        'storage_mode': lt.storage_mode_t(2),
    }
    handle = lt.add_magnet_uri(ses, magnet_link, params)
    handle.pause()
    console.print("[yellow]Fetching metadata...[/yellow]")

    while not handle.has_metadata():
        time.sleep(1)

    show_metadata(handle)

    if Confirm.ask("Do you want to download this torrent?"):
        send_to_daemon({
            "type": "magnet",
            "magnet_link": magnet_link,
            "save_path": save_path
        })
    else:
        console.print("[red]Download cancelled.[/red]")


def download_from_torrent_file(torrent_path, save_path):
    ses = lt.session()
    ses.listen_on(6881, 6891)
    info = lt.torrent_info(torrent_path)
    params = {
        'save_path': '/tmp',
        'ti': info
    }
    handle = ses.add_torrent(params)
    handle.pause()

    show_metadata(handle)

    if Confirm.ask("Do you want to download this torrent?"):
        send_to_daemon({
            "type": "torrent_file",
            "torrent_path": torrent_path,
            "save_path": save_path
        })
    else:
        console.print("[red]Download cancelled.[/red]")
