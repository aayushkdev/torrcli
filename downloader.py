import json
import os
import asyncio
from rich.console import Console
from rich.table import Table
from rich.prompt import Confirm
from rich.progress import Progress

SOCKET_PATH = "/tmp/torrcli_daemon.sock"
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
        metadata["size_mb"],
        str(metadata["num_files"]),
        str(metadata.get("seeders", "N/A")),
        str(metadata.get("leechers", "N/A")),
    )

    console.print(table)


async def send_to_daemon(data):
    if not os.path.exists(SOCKET_PATH):
        console.print("[red]Daemon is not running![/red]")
        return None

    reader, writer = await asyncio.open_unix_connection(SOCKET_PATH)
    writer.write(json.dumps(data).encode())
    await writer.drain()

    raw = await reader.read(4096)
    writer.close()
    await writer.wait_closed()

    response = json.loads(raw.decode())

    if response.get("status") == "metadata":
        return response["data"]
    else:
        console.print(f"[red]Daemon response:[/red] {response}")
        return None


async def progress(source):
    if not os.path.exists(SOCKET_PATH):
        console.print("[red]Daemon is not running![/red]")
        return

    reader, writer = await asyncio.open_unix_connection(SOCKET_PATH)

    writer.write(json.dumps({
        "type": "get_progress",
        "source": source
    }).encode())
    await writer.drain()

    buffer = ""
    with Progress() as progress:
        task = progress.add_task("[cyan]Downloading...", total=100)
        while not reader.at_eof():
            chunk = await reader.read(4096)
            if not chunk:
                break
            buffer += chunk.decode()
            while "\n" in buffer:
                line, buffer = buffer.split("\n", 1)
                try:
                    data = json.loads(line)
                    if data["status"] in ("downloading", "seeding"):
                        progress.update(task, completed=data["download_progress"])
                        if data["status"] == "seeding":
                            console.print("[green]Download complete![/green]")
                            writer.close()
                            await writer.wait_closed()
                            return
                except json.JSONDecodeError:
                    continue


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
