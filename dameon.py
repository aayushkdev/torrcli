import sys
import json
import signal
import asyncio
from pathlib import Path
import libtorrent as lt

PID_FILE = "/tmp/torrcli_daemon.pid"
SOCKET_PATH = "/tmp/torrcli_daemon.sock"

ses = lt.session({'listen_interfaces': '0.0.0.0:6881'})
torrent_handles = {}

def clean_exit(*_):
    Path(SOCKET_PATH).unlink(missing_ok=True)
    sys.exit(0)

async def handle_request(reader, writer):
    try:
        data = await reader.read(4096)
        if not data:
            writer.close()
            await writer.wait_closed()
            return

        request = json.loads(data.decode())
        req_type = request.get("type")
        source = request.get("source")
        save_path = request.get("save_path")

        if req_type == "add_torrent":
            if source.startswith("magnet:"):
                handle = ses.add_torrent({"url": source, "save_path": save_path})
                torrent_handles[source] = handle
                while not handle.has_metadata():
                    await asyncio.sleep(1)
                handle.pause()
                ti = handle.get_torrent_info()
            else:
                info = lt.torrent_info(source)
                handle = ses.add_torrent({"ti": info, "save_path": save_path})
                handle.pause()
                torrent_handles[source] = handle
                ti = info

            status = handle.status()
            metadata = {
                "name": ti.name(),
                "size_bytes": ti.total_size(),
                "num_files": ti.num_files(),
                "seeders": status.num_seeds,
                "leechers": status.num_peers - status.num_seeds,
            }
            writer.write(json.dumps({"status": "metadata", "data": metadata}).encode())

        elif req_type == "start_download":
            handle = torrent_handles.get(source)
            if handle:
                if not handle.status().paused:
                    writer.write(b'{"status": "already_resumed"}')
                else:
                    handle.resume()
                    writer.write(b'{"status": "resumed"}')
            else:
                writer.write(b'{"status": "error", "message": "Torrent not found"}')

        elif req_type == "pause_download":
            handle = torrent_handles.get(source)
            if handle:
                if handle.status().paused:
                    writer.write(b'{"status": "already_paused"}')
                else:
                    handle.auto_managed(False)
                    handle.pause()
                    writer.write(b'{"status": "paused"}')
            else:
                writer.write(b'{"status": "error", "message": "Torrent not found"}')

        elif req_type == "remove_download":
            handle = torrent_handles.pop(source, None)
            if handle:
                ses.remove_torrent(handle)
                writer.write(b'{"status": "removed"}')
            else:
                writer.write(b'{"status": "error", "message": "Torrent not found"}')

        elif req_type == "get_progress":
            handle = torrent_handles.get(source)
            if handle:
                status = handle.status()
                remaining = status.total_wanted - status.total_wanted_done
                time_left = remaining / status.download_rate if status.download_rate > 0 else 0

                progress_data = {
                    "name": handle.name(),
                    "status": "downloading" if not status.is_seeding else "seeding",
                    "download_progress": status.progress * 100,
                    "time_left": time_left,
                    "seeders": status.num_seeds,
                    "leechers": max(status.num_peers - status.num_seeds, 0),
                    "peers": status.num_peers,
                    "download_speed": status.download_rate,  
                    "downloaded_bytes": status.total_wanted_done,
                    "total_bytes": max(status.total_wanted, 1),

                }
                writer.write((json.dumps(progress_data)).encode())
            else:
                writer.write(b'{"status": "error", "message": "Torrent not found"}')


        else:
            writer.write(b'{"status": "error", "message": "Unknown request type"}')

        await writer.drain()
        writer.close()
        await writer.wait_closed()

    except Exception as e:
        print(f"Error: {e}")
        writer.close()

async def socket_server():
    Path(SOCKET_PATH).unlink(missing_ok=True)
    server = await asyncio.start_unix_server(handle_request, path=SOCKET_PATH)

    async with server:
        await server.serve_forever()

def main():
    signal.signal(signal.SIGTERM, clean_exit)
    signal.signal(signal.SIGINT, clean_exit)
    asyncio.run(socket_server())

if __name__ == "__main__":
    main()