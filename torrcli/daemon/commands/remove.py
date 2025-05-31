from torrcli.daemon.commands.utils import send_response, send_error
from torrcli.daemon.session import torrent_handles, ses
from torrcli.daemon.config import TORRENT_DATA_DIR

async def handle(request, writer):
    info_hash = request.get("source")
    handle = torrent_handles.pop(info_hash, None)

    if not handle:
        return await send_error(writer, "Torrent not found")

    ses.remove_torrent(handle)

    fastresume_file = TORRENT_DATA_DIR / f"{info_hash}.fastresume"
    torrent_file = TORRENT_DATA_DIR / f"{info_hash}.torrent"
    for f in (fastresume_file, torrent_file):
        if f.exists():
            f.unlink()

    await send_response(writer, {"status": "removed"})