import sys
import tty
import termios
import selectors    
import json
import os
import asyncio
from rich.console import Console, Group
from rich.table import Table
from rich.prompt import Confirm
from rich.live import Live
from rich.text import Text
from rich.progress import Progress, BarColumn, TextColumn
from rich.panel import Panel
from rich.align import Align

SOCKET_PATH = "/tmp/torrcli_daemon.sock"
console = Console()

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


def read_key():
    fd = sys.stdin.fileno()
    old_settings = termios.tcgetattr(fd)
    try:
        tty.setraw(fd)
        return sys.stdin.read(1)
    finally:
        termios.tcsetattr(fd, termios.TCSADRAIN, old_settings)


def show_metadata(metadata):
    table = Table(title="Torrent Metadata", show_lines=True)
    table.add_column("Name", style="bold cyan")
    table.add_column("Size", style="green")
    table.add_column("Files", style="green")
    table.add_column("Seeders", style="yellow")
    table.add_column("Leechers", style="yellow")

    table.add_row(
        metadata["name"],
        format_size(metadata["size_bytes"]),
        str(metadata["num_files"]),
        str(metadata.get("seeders", "N/A")),
        str(metadata.get("leechers", "N/A")),
    )

    console.print(table)


async def send_command(data):
    if not os.path.exists(SOCKET_PATH):
        console.print("[red]Daemon is not running![/red]")
        return None, None

    reader, writer = await asyncio.open_unix_connection(SOCKET_PATH)
    writer.write(json.dumps(data).encode())
    await writer.drain()
    return reader, writer

async def send_and_receive(data):
    reader, writer = await send_command(data)
    if reader is None or writer is None:
        return None

    raw = await reader.read(4096)
    writer.close()
    await writer.wait_closed()

    try:
        return json.loads(raw.decode())
    except json.JSONDecodeError:
        return None

async def send_to_daemon(data):
    response = await send_and_receive(data)

    if response.get("status") == "metadata":
        return response["data"]
    elif response.get("status") == "error":
        console.print(f"[red]Daemon response:[/red] {response}")
        return None
    else:
        return None


def setup_nonblocking_input():
    fd = sys.stdin.fileno()
    tty.setcbreak(fd) 
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

async def progress(source):
    console.print("[dim]Press [bold]p[/bold] to pause, [bold]r[/bold] to resume, [bold]e[/bold] to exit[/dim]\n")

    status_label = {"text": "Downloading"}
    state_event = asyncio.Event()
    exit_event = asyncio.Event()

    async def handle_keys():
        setup_nonblocking_input()
        try:
            while not exit_event.is_set():
                key = get_pressed_key()
                if key:
                    if key == "p":
                        await send_and_receive({"type": "pause_download", "source": source})
                        status_label["text"] = "Paused"
                        state_event.set()
                    elif key == "r":
                        await send_and_receive({"type": "start_download", "source": source})
                        status_label["text"] = "Downloading"
                        state_event.set()
                    elif key == "e":
                        exit_event.set()
                await asyncio.sleep(0.1) 
        finally:
            restore_input_mode()

    progress_bar = Progress(
        TextColumn("[bold blue]{task.description}"),
        BarColumn(),
        TextColumn("[green]{task.percentage:>3.1f}%"),
    )
    task_id = progress_bar.add_task(f"{status_label['text']}...", total=100)

    
    def render_ui(name, completed_bytes, total_bytes, eta_sec, progress, speed_bps, seeders, leechers):
        name_text = Text(name, style="bold underline magenta")

        eta_line = Text(
            f"{format_size(completed_bytes)} of {format_size(total_bytes)} — {format_time(eta_sec)} left",
            style="cyan"
        )

        peer_speed_line = Text(
            f"Seeders: {seeders} | Leechers: {leechers} | Download Speed: {format_speed(speed_bps)}",
            style="green"
        )

        group = Group(
            name_text,
            Text(""),
            eta_line,
            Text(""),
            progress,
            Text(""),
            peer_speed_line
        )

        panel = Panel(
            Align.center(group),
            title="[bold yellow]Torrent Download[/bold yellow]",
            border_style="bright_blue",
            padding=(1, 2),
            expand=False,
        )

        return panel



    asyncio.create_task(handle_keys())

    with Live(render_ui("Loading...", 0, 1, 0, progress_bar, 0, 0, 0), refresh_per_second=4, console=console) as live:
            while not exit_event.is_set():
                data = await send_and_receive({
                    "type": "get_progress",
                    "source": source
                })

                if not data or "status" not in data:
                    break

                if data["status"] in ("downloading", "seeding"):
                    progress_bar.update(task_id, completed=data["download_progress"])
                    if state_event.is_set():
                        progress_bar.update(task_id, description=f"{status_label['text']}...")
                        state_event.clear()

                    live.update(render_ui(
                        name=data.get("name", "Unknown Torrent"),
                        completed_bytes=data.get("downloaded_bytes", 0),
                        total_bytes=data.get("total_bytes", 1),
                        eta_sec=data.get("time_left", 0),
                        progress=progress_bar,
                        speed_bps=data.get("download_speed", 0),
                        seeders=data.get("seeders", 0),
                        leechers=data.get("leechers", 0),
                    ))


                    if data["status"] == "seeding":
                        console.print("[green]Download complete![/green]")
                        exit_event.set()
                        break

                await asyncio.sleep(1)


async def download(source, save_path):
    metadata = await send_to_daemon({
        "type": "add_torrent",
        "source": source,
        "save_path": save_path
    })

    if metadata:
        show_metadata(metadata)

        if Confirm.ask("Do you want to download this torrent?"):
            await send_to_daemon({
                "type": "start_download",
                "source": source,
                "save_path": save_path
            })

            await progress(source)
        else:
            console.print("[red]Download cancelled. Removing torrent...[/red]")
            await send_to_daemon({
                "type": "remove_download",
                "source": source
            })
