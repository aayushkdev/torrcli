import asyncio
from ui import console, show_metadata, render_ui
from client import send_to_daemon, send_and_receive
from utils import setup_nonblocking_input, restore_input_mode, get_pressed_key, format_size, format_time
from rich.live import Live
from rich.prompt import Confirm
from rich.progress import Progress, TextColumn, BarColumn
from rich.table import Table

async def progress(source):
    console.print("[dim]Press [bold]p[/bold] to pause, [bold]r[/bold] to resume, [bold]e[/bold] to exit[/dim]\n")

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
                        state_event.set()
                    elif key == "r":
                        await send_and_receive({"type": "start_download", "source": source})
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

    init_data = await send_and_receive({
        "type": "get_progress",
        "source": source
    })

    state = init_data.get("state", "loading").capitalize()
    task_id = progress_bar.add_task(f"{state}...", total=100)

    asyncio.create_task(handle_keys())

    with Live(render_ui("Loading...", 0, 1, 0, progress_bar, 0, 0, 0, state), refresh_per_second=4, console=console) as live:
        while not exit_event.is_set():
            data = await send_and_receive({
                "type": "get_progress",
                "source": source
            })

            if not data or "state" not in data:
                break

            state = data["state"].capitalize()
            progress_bar.update(task_id, completed=data["download_progress"])
            progress_bar.update(task_id, description=f"{state}...")

            live.update(render_ui(
                name=data.get("name", "Unknown Torrent"),
                completed_bytes=data.get("downloaded_bytes", 0),
                total_bytes=data.get("total_bytes", 1),
                eta_sec=data.get("time_left", 0),
                progress=progress_bar,
                speed_bps=data.get("download_speed", 0),
                seeders=data.get("seeders", 0),
                leechers=data.get("leechers", 0),
                status_text=state
            ))

            if data["state"] == "seeding":
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


async def list_torrents():
    response = await send_and_receive({"type": "list_torrents"})

    if not response or response.get("status") != "success":
        console.print("[red]No torrents found or failed to connect.[/red]")
        return

    torrents = response.get("data", [])

    table = Table(title="Current Torrents", show_lines=True)
    table.add_column("Name", style="cyan", overflow="fold")
    table.add_column("Progress", justify="right", style="green")
    table.add_column("State", style="yellow")
    table.add_column("Size", justify="right")
    table.add_column("ETA", justify="right")

    for t in torrents:
        table.add_row(
            t["name"],
            f"{t['progress']}%",
            t["state"],
            f"{format_size(t["downloaded"])}/{format_size(t["total_size"])}",
            format_time(t["time_left"]),
        )

    console.print(table)
