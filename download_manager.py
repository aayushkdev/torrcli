import asyncio
from ui import console, show_metadata, render_ui
from client import send_to_daemon, send_and_receive
from utils import setup_nonblocking_input, restore_input_mode, get_pressed_key
from rich.live import Live
from rich.prompt import Confirm
from rich.progress import Progress, TextColumn, BarColumn

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

    asyncio.create_task(handle_keys())


    with Live(render_ui("Loading...", 0, 1, 0, progress_bar, 0, 0, 0, status_label['text']), refresh_per_second=4, console=console) as live:
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
                    status_text=status_label["text"]
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
