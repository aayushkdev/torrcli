from rich.console import Console, Group
from rich.table import Table
from rich.text import Text
from rich.panel import Panel
from rich.align import Align
from utils import format_size, format_speed, format_time

console = Console()

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

def render_ui(name, completed_bytes, total_bytes, eta_sec, progress, speed_bps, seeders, leechers, status_text):
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
        title=f"[bold yellow]Torrent Download - {status_text}[/bold yellow]",
        border_style="bright_blue",
        padding=(1, 2),
        expand=False,
    )

    return panel
