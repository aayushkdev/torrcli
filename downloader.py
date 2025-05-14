import json
import socket
import os
from rich.console import Console
from rich.table import Table
from rich.prompt import Confirm

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

def send_to_daemon(data):
    if not os.path.exists(SOCKET_PATH):
        console.print("[red]Daemon is not running![/red]")
        return None

    with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as sock:
        sock.connect(SOCKET_PATH)
        sock.sendall(json.dumps(data).encode())
        response = sock.recv(4096).decode()
        response_data = json.loads(response)

        if response_data.get("status") == "metadata":
            return response_data["data"]
        else:
            console.print(f"[red]Daemon response:[/red] {response}")
            return None

def download(source, save_path):
    metadata = send_to_daemon({
        "type": "add_torrent",
        "source": source,
        "save_path": save_path
    })

    if metadata:
        show_metadata(metadata)

        if Confirm.ask("Do you want to download this torrent?"):
            send_to_daemon({
                "type": "start_download",
                "source": source,
                "save_path": save_path
            })
        else:
            console.print("[red]Download cancelled. Pausing torrent...[/red]")
            send_to_daemon({
                "type": "remove_download",
                "source": source
            })
