import argparse
import os
import asyncio
from pathlib import Path
from torrcli.client.commands.download import download
from torrcli. client.commands.listings import list_torrents, info
from torrcli.client.commands.manage import pause, resume, remove

def is_magnet_link(value: str) -> bool:
    return value.startswith("magnet:?xt=urn:")

def is_torrent_file(path: str) -> bool:
    return os.path.isfile(path) and path.endswith(".torrent")

async def main():
    parser = argparse.ArgumentParser(description="Torrent CLI Downloader")
    subparsers = parser.add_subparsers(dest="command")

    download_parser = subparsers.add_parser("download", help="Download a torrent")
    download_parser.add_argument("source", help="Magnet link or .torrent file")
    download_parser.add_argument("--save", help="Save path", default=str(Path.home() / "Downloads"))

    subparsers.add_parser("list", help="List all torrents")

    info_parser = subparsers.add_parser("info", help="Show detailed info of a torrent")
    info_parser.add_argument("index", type=int, help="Index of the torrent")

    pause_parser = subparsers.add_parser("pause", help="Pause a torrent")
    pause_parser.add_argument("index", type=int, help="Index of the torrent")

    resume_parser = subparsers.add_parser("resume", help="Resume a torrent")
    resume_parser.add_argument("index", type=int, help="Index of the torrent")

    remove_parser = subparsers.add_parser("remove", help="Remove a torrent")
    remove_parser.add_argument("index", type=int, help="Index of the torrent")

    args = parser.parse_args()


    if args.command == "list":
        await list_torrents()

    elif args.command == "download":
        if is_magnet_link(args.source) or is_torrent_file(args.source):
            await download(args.source, args.save)
        else:
            print("Error: Not a valid magnet link or .torrent file path.")

    elif args.command == "info":
        await info(args.index)

    elif args.command == "pause":
        await pause(args.index)

    elif args.command == "resume":
        await resume(args.index)

    elif args.command == "remove":
        await remove(args.index)

    else:
        parser.print_help()

def run():
    asyncio.run(main())

if __name__ == "__main__":
    run()