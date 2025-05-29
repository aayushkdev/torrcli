from torrcli.client.ipc import send_and_receive
from torrcli.client.ui import console
from rich.table import Table
from torrcli.client.utils import format_size, format_time
from torrcli.client.commands.progress import progress
from torrcli.client.commands.manage import resolve_info_hash


async def info(index):
    info_hash = await resolve_info_hash(index)
    if info_hash:
        await progress(info_hash)


async def list_torrents():
    response = await send_and_receive({"type": "list_torrents"})

    torrents = response.get("data", [])

    if not torrents:
        console.print("[bold yellow]No torrents added yet.[/bold yellow]")
        console.print("Use [green]torrcli download <magnet or .torrent file>[/green] to add a torrent.")
        return

    table = Table(title="Current Torrents", show_lines=True)
    table.add_column("#", justify="right")
    table.add_column("Name", style="cyan", overflow="fold")
    table.add_column("Progress", justify="right", style="green")
    table.add_column("State", style="yellow")
    table.add_column("Size", justify="right")
    table.add_column("ETA", justify="right")

    for t in torrents:
        table.add_row(
            str(t["index"]),
            t["name"],
            f"{t['progress']}%",
            t["state"],
            f"{format_size(t["downloaded"])}/{format_size(t["total_size"])}",
            format_time(t["time_left"]),
        )

    console.print(table)
