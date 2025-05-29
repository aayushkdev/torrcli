from torrcli.client.ipc import send_and_receive
from torrcli.client.ui import console


async def resolve_info_hash(index):
    response = await send_and_receive({"type": "list_torrents"})
    if not response["data"]:
        console.print(f"[red]Invalid index: {index}[/red]")
        return
    torrents = response.get("data", [])
    index_map = {t["index"]: t["info_hash"] for t in torrents}
    return index_map.get(index)


async def pause(index):
    info_hash = await resolve_info_hash(index)
    if info_hash:
        await send_and_receive({"type": "pause_download", "source": info_hash})
        console.print(f"[yellow]Torrent at index {index} paused.[/yellow]")


async def resume(index):
    info_hash = await resolve_info_hash(index)
    if info_hash:
        await send_and_receive({"type": "start_download", "source": info_hash})
        console.print(f"[green]Torrent at index {index} resumed.[/green]")


async def remove(index):
    info_hash = await resolve_info_hash(index)
    if info_hash:
        await send_and_receive({"type": "remove_download", "source": info_hash})
        console.print(f"[red]Torrent at index {index} removed.[/red]")