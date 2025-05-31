from torrcli.daemon.commands.utils import send_error, send_success, calc_eta, get_torrent_state
from torrcli.daemon.session import torrent_handles

async def handle(request, writer):
    info_hash = request.get("source")
    handle = torrent_handles.get(info_hash)

    if not handle:
        return await send_error(writer, "Torrent not found")

    status = handle.status()
    eta = calc_eta(status)

    progress_data = {
        "name": handle.name(),
        "progress": round(status.progress * 100, 2),
        "downloaded": status.total_done,
        "size": max(status.total_wanted, 1),
        "eta": eta,
        "download_speed": status.download_rate,
        "upload_speed": status.upload_rate,
        "seeders": status.num_seeds,
        "leechers": max(status.num_peers - status.num_seeds, 0),
        "state": get_torrent_state(handle),
    }
    await send_success(writer, progress_data)