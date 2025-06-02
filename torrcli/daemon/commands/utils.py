import json
import math
import libtorrent as lt

async def send_response(writer, response):
    writer.write(json.dumps(response).encode())
    await writer.drain()
    writer.close()
    await writer.wait_closed()

async def send_error(writer, message):
    await send_response(writer, {"status": "error", "message": message})

async def send_success(writer, data):
    await send_response(writer, {"status": "success", "data": data})

def calc_eta(status):
    rate = status.download_rate

    if rate < 10:
        return -1
    else:
        remaining = status.total_wanted - status.total_done
        return math.ceil(remaining / rate)

def get_torrent_state(handle):
    if handle.is_paused():
        return "paused"

    status = handle.status()

    state_map = {
        lt.torrent_status.queued_for_checking: "queued for checking",
        lt.torrent_status.checking_files: "checking",
        lt.torrent_status.downloading_metadata: "downloading metadata",
        lt.torrent_status.downloading: "downloading",
        lt.torrent_status.finished: "finished",
        lt.torrent_status.seeding: "seeding",
        lt.torrent_status.allocating: "allocating",
        lt.torrent_status.checking_resume_data: "checking resume data",
    }

    return state_map.get(status.state, "unknown")