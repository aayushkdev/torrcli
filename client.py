import os
import json
import asyncio
from ui import console

SOCKET_PATH = "/tmp/torrcli_daemon.sock"

async def send_command(data):
    if not os.path.exists(SOCKET_PATH):
        console.print("[red]Daemon is not running![/red]")
        return None, None

    reader, writer = await asyncio.open_unix_connection(SOCKET_PATH)
    writer.write(json.dumps(data).encode())
    await writer.drain()
    return reader, writer

async def send_and_receive(data):
    reader, writer = await send_command(data)
    if reader is None or writer is None:
        return None

    raw = await reader.read(4096)
    writer.close()
    await writer.wait_closed()

    try:
        return json.loads(raw.decode())
    except json.JSONDecodeError:
        return None

async def send_to_daemon(data):
    response = await send_and_receive(data)

    if response is None:
        return None

    if response.get("status") == "metadata":
        return response["data"]
    elif response.get("status") == "error":
        console.print(f"[red]Daemon response:[/red] {response}")
        return None
    else:
        return response
