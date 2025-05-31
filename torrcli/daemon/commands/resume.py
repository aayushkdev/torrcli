from torrcli.daemon.commands.utils import send_response, send_error
from torrcli.daemon.session import torrent_handles

async def handle(request, writer):
    info_hash = request.get("source")
    handle = torrent_handles.get(info_hash)

    if not handle:
        return await send_error(writer, "Torrent not found")

    if not handle.is_paused():
        return await send_response(writer, {"status": "already_resumed"})

    handle.resume()
    await send_response(writer, {"status": "resumed"})