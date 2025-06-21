from rich.console import Console, Group
from rich.table import Table
from rich.text import Text
from rich.panel import Panel
from rich.align import Align
from torrcli.client.utils import format_size, format_speed, format_time

console = Console()

def show_metadata(metadata):
    table = Table(title="Torrent Metadata", show_lines=True)
    table.add_column("Field", style="bold cyan")
    table.add_column("Value", style="green")

    table.add_row("Name", metadata["name"])
    table.add_row("Size", format_size(metadata["size_bytes"]))
    table.add_row("Files", str(metadata["num_files"]))
    table.add_row("Pieces", str(metadata.get("num_pieces", "N/A")))
    table.add_row("Piece Size", format_size(metadata.get("piece_length", 0)))
    table.add_row("Info Hash", metadata.get("info_hash", "N/A"))
    files = metadata.get("files", [])
    if files:
        files_sorted = sorted(files, key=lambda f: f["size"], reverse=True)
        file_lines = [f"{f['path']} ({format_size(f['size'])})" for f in files_sorted]
        table.add_row("File List", "\n".join(file_lines))

    console.print(table)


def render_ui(name, completed_bytes, total_bytes, eta_sec, progress, download_speed, upload_speed, seeders, leechers, connected_peers, total_peers, status_text):
    name_text = Text(name, style="bold underline magenta")

    eta_line = Text(
        f"{format_size(completed_bytes)} of {format_size(total_bytes)} — {format_time(eta_sec)} left",
        style="cyan"
    )

    peer_speed_line = Text(
        f"S: {seeders} | L: {leechers} | P: {connected_peers}({total_peers}) | DL: {format_speed(download_speed)} | UL: {format_speed(upload_speed)}",
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


def show_torrent_search_results(results, query):
    console.print(f"[bold underline]Search Results for: {query}[/bold underline]\n")

    total = len(results)
    reversed_results = list(reversed(results))

    max_title_len = 80
    index_width = len(str(total))

    for i, r in enumerate(reversed_results):
        idx = total - i
        idx_str = f"{idx:<{index_width}}"

        title = r['title'][:max_title_len]
        size = r['size']
        seeders = r['seeders']
        leechers = r['leechers']
        source = r['source']

        console.print(
            f"[bold cyan]{idx_str}[/] [white]{title}[/white]"
        )

        console.print(
            f"{' ' * (index_width + 1)}"
            f"  [green]{size}[/green] "
            f"[yellow]{seeders} seeders[/yellow], "
            f"[red]{leechers} leechers[/red] "
            f"— [dim]{source}[/dim]"
        )
