import os
import json
import asyncio
import configparser
from pathlib import Path
from torrcli.client.ui import console

_conf_path = Path.home() / ".config" / "torrcli" / "torrcli.conf"
_conf = configparser.ConfigParser()
_conf.read(str(_conf_path))
SOCKET_PATH = _conf.get("general", "socket_path", fallback="/tmp/torrcli_daemon.sock")

async def send_command(data):
    if not os.path.exists(SOCKET_PATH):
        console.print("[red]Daemon is not running![/red]")
        return None, None

    reader, writer = await asyncio.open_unix_connection(SOCKET_PATH)
    writer.write(json.dumps(data, separators=(",", ":")).encode() + b"\n")
    await writer.drain()
    return reader, writer

async def send_and_receive(data, timeout=30):
    try:
        reader, writer = await send_command(data)
    except OSError as exc:
        console.print(f"[red]Failed to connect to daemon: {exc}[/red]")
        return None

    if reader is None or writer is None:
        return None

    try:
        if timeout is not None:
            raw = await asyncio.wait_for(reader.readline(), timeout=timeout)
        else:
            raw = await reader.readline()
    except asyncio.TimeoutError:
        console.print("[red]Timed out waiting for daemon response.[/red]")
        return None
    except Exception as exc:
        console.print(f"[red]Connection error reading response: {exc}[/red]")
        return None
    finally:
        try:
            writer.close()
            await writer.wait_closed()
        except Exception:
            pass

    try:
        return json.loads(raw.decode())
    except json.JSONDecodeError:
        return None
