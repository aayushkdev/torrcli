import libtorrent as lt
import time
import os
from rich.console import Console
from rich.progress import Progress
from rich.prompt import Confirm

console = Console()


def _download_torrent(handle):
    with Progress() as progress:
        task = progress.add_task("[cyan]Downloading...", total=100)
        while not handle.is_seed():
            s = handle.status()
            progress.update(task, completed=s.progress * 100)
            time.sleep(1)
        console.print("[green]Download complete![/green]")


def download_from_magnet(magnet_link, save_path="./downloads"):
    ses = lt.session()
    ses.listen_on(6881, 6891)
    params = {
        'save_path': save_path,
        'storage_mode': lt.storage_mode_t(2),
    }
    handle = lt.add_magnet_uri(ses, magnet_link, params)
    console.print("[yellow]Fetching metadata...[/yellow]")

    while not handle.has_metadata():
        time.sleep(1)

    show_metadata(handle)

    if Confirm.ask("Do you want to download this torrent?"):
        _download_torrent(handle)
    else:
        console.print("[red]Download cancelled.[/red]")


def download_from_torrent_file(torrent_path, save_path="./downloads"):
    ses = lt.session()
    ses.listen_on(6881, 6891)
    info = lt.torrent_info(torrent_path)
    params = {
        'save_path': save_path,
        'ti': info
    }
    handle = ses.add_torrent(params)

    show_metadata(handle)

    if Confirm.ask("Do you want to download this torrent?"):
        _download_torrent(handle)
    else:
        console.print("[red]Download cancelled.[/red]")
