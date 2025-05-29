from rich.prompt import Confirm
from torrcli.client.ipc import send_and_receive
from torrcli.client.ui import console, show_metadata
from torrcli.client.commands.progress import progress

async def download(source, save_path):
    response = await send_and_receive({
        "type": "add_torrent",
        "source": source,
        "save_path": save_path
    })
    if response["status"] == "success":
        metadata = response["data"]
        show_metadata(metadata)

        if Confirm.ask("Do you want to download this torrent?"):
            await send_and_receive({
                "type": "start_download",
                "source": metadata["info_hash"],
            })

            await progress(metadata["info_hash"])
        else:
            console.print("[red]Download cancelled. Removing torrent...[/red]")
            await send_and_receive({
                "type": "remove_download",
                "source": metadata["info_hash"],
            })
    else:
        console.print(f"[red]{response["message"]}[/red]")