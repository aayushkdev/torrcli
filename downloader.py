import libtorrent as lt
import time
import os
from rich.console import Console
from rich.progress import Progress
from rich.table import Table
from rich.prompt import Confirm

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


def _download_torrent(handle):
    if handle.is_seed():
        s = handle.status()
        console.print(f"[green]Torrent already downloaded at:[/green] {s.save_path}")
        return

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
    handle.pause()
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
    handle.pause()

    show_metadata(handle)

    if Confirm.ask("Do you want to download this torrent?"):
        _download_torrent(handle)
    else:
        console.print("[red]Download cancelled.[/red]")
