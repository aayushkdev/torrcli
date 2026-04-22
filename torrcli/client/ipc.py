import os
import json
import asyncio
from torrcli.client.ui import console

SOCKET_PATH = "/tmp/torrcli_daemon.sock"

async def send_command(data):
    if not os.path.exists(SOCKET_PATH):
        console.print("[red]Daemon is not running![/red]")
        return None, None

    reader, writer = await asyncio.open_unix_connection(SOCKET_PATH)
    writer.write(json.dumps(data, separators=(",", ":")).encode() + b"\n")
    await writer.drain()
    return reader, writer

async def send_and_receive(data):
    try:
        reader, writer = await send_command(data)
    except OSError as exc:
        console.print(f"[red]Failed to connect to daemon: {exc}[/red]")
        return None

    if reader is None or writer is None:
        return None

    try:
        raw = await asyncio.wait_for(reader.readline(), timeout=30)
    except asyncio.TimeoutError:
        console.print("[red]Timed out waiting for daemon response.[/red]")
        return None
    finally:
        writer.close()
        await writer.wait_closed()

    try:
        return json.loads(raw.decode())
    except json.JSONDecodeError:
        return None
