import asyncio
import json
from pathlib import Path
from torrcli.daemon.alerts import alert_loop
from torrcli.daemon.config import SOCKET_PATH
from torrcli.daemon.commands import add, pause, resume, remove, list, progress

async def handle_request(reader, writer):
    request = json.loads((await reader.read(4096)).decode())
    req_type = request.get("type")

    handlers = {
        "add_torrent": add.handle,
        "pause_download": pause.handle,
        "start_download": resume.handle,
        "remove_download": remove.handle,
        "list_torrents": list.handle,
        "get_progress": progress.handle,
    }

    handler = handlers.get(req_type)
    if handler:
        await handler(request, writer)
    else:
        writer.write(b'{"status": "error", "message": "Unknown request type"}')
        await writer.drain()
        writer.close()

async def socket_server():
    Path(SOCKET_PATH).unlink(missing_ok=True)
    server = await asyncio.start_unix_server(handle_request, path=SOCKET_PATH)
    print(f"Daemon listening on {SOCKET_PATH}")

    async with server:
        alert_task = asyncio.create_task(alert_loop())
        serve_task = asyncio.create_task(server.serve_forever())

        await alert_task

        server.close()
        await server.wait_closed()

        serve_task.cancel()
        try:
            await serve_task
        except asyncio.CancelledError:
            pass