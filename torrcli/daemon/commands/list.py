from torrcli.daemon.commands.utils import send_success, calc_eta, get_torrent_state
from torrcli.daemon.session import ses

async def handle(request, writer):
    torrents = []
    for i, handle in enumerate(ses.get_torrents()):
        status = handle.status()
        torrents.append({
            "index": i + 1,
            "name": status.name,
            "progress": round(status.progress * 100, 2),
            "state": get_torrent_state(handle),
            "downloaded": status.total_done,
            "size": max(status.total_wanted, 1),
            "eta": calc_eta(status),
            "info_hash": str(status.info_hashes.get_best())
        })

    await send_success(writer, torrents)