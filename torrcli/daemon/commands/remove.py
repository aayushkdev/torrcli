from torrcli.daemon.commands.utils import send_response, send_error
from torrcli.daemon.session import torrent_handles, ses
from torrcli.daemon.config import DATA_DIR

async def remove_torrent(info_hash):
    handle = torrent_handles.pop(info_hash, None)
    if handle:
        ses.remove_torrent(handle)

        fastresume_file = DATA_DIR / f"{info_hash}.fastresume"
        torrent_file = DATA_DIR / f"{info_hash}.torrent"
        for f in (fastresume_file, torrent_file):
            if f.exists():
                f.unlink()

        return True
    return False

async def handle(request, writer):
    info_hash = request.get("source")
    removed = await remove_torrent(info_hash)
    if not removed:
        return await send_error(writer, "Torrent not found")
    await send_response(writer, {"status": "removed"})