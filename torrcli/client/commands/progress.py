import asyncio
from rich.live import Live
from rich.progress import Progress, TextColumn, BarColumn
from torrcli.client.ipc import send_and_receive
from torrcli.client.ui import console, render_ui
from torrcli.client.utils import setup_nonblocking_input, restore_input_mode, get_pressed_key

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

    init_response = await send_and_receive({
        "type": "get_progress",
        "source": source
    })

    if not init_response or init_response.get("status") != "success":
        console.print("[red]Failed to fetch initial progress.[/red]")
        return

    init_data = init_response["data"]
    state = init_data.get("state", "loading").capitalize()
    task_id = progress_bar.add_task(f"{state}...", total=100)

    asyncio.create_task(handle_keys())

    with Live(render_ui("Loading...", 0, 1, 0, progress_bar, 0, 0, 0, 0, 0, 0, state), refresh_per_second=4, console=console) as live:
        while not exit_event.is_set():
            response = await send_and_receive({
                "type": "get_progress",
                "source": source
            })

            if not response or response.get("status") != "success":
                break

            data = response["data"]

            state = data["state"].capitalize()
            progress_bar.update(task_id, completed=data["progress"])
            progress_bar.update(task_id, description=f"{state}...")

            live.update(render_ui(
                name=data.get("name", "Unknown Torrent"),
                completed_bytes=data.get("downloaded", 0),
                total_bytes=data.get("size", 1),
                eta_sec=data.get("eta", 0),
                progress=progress_bar,
                download_speed=data.get("download_speed", 0),
                upload_speed=data.get("upload_speed", 0),
                seeders=data.get("seeders", 0),
                leechers=data.get("leechers", 0),
                connected_peers=data.get("connected_peers", 0),
                total_peers=data.get("total_peers", 0),
                status_text=state,
            ))

            await asyncio.sleep(1)

