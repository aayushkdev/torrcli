import asyncio
import libtorrent as lt
from pathlib import Path
from torrcli.daemon.config import DATA_DIR, DEFAULT_SAVE_PATH
from torrcli.daemon.session import ses, torrent_handles
from torrcli.daemon.commands.utils import send_success, send_error

async def handle(request, writer):
    try:
        source = request.get("source")
        save_path = request.get("save_path") or str(DEFAULT_SAVE_PATH)
        stream = request.get("stream", False)
        if source.startswith("magnet:"):
            handle = ses.add_torrent({"url": source, "save_path": save_path})
            while not handle.has_metadata():
                await asyncio.sleep(1)

            ti = handle.get_torrent_info()
        else:
            ti = lt.torrent_info(source)

        info_hash = str(ti.info_hash())
        torrent_handles[info_hash] = handle if source.startswith("magnet:") else ses.add_torrent({
            "ti": ti,
            "save_path": save_path,
            **({"resume_data": lt.bdecode((DATA_DIR / f"{info_hash}.fastresume").read_bytes())}
               if (DATA_DIR / f"{info_hash}.fastresume").exists() else {})
        })

        handle = torrent_handles[info_hash]
        handle.pause()
        handle.auto_managed(False)
        handle.save_resume_data()

        if stream:
            handle.set_sequential_download(True)

        torrent_path = DATA_DIR / f"{info_hash}.torrent"
        if not torrent_path.exists():
            if source.startswith("magnet:"):
                t = lt.create_torrent(ti)
                torrent_path.write_bytes(lt.bencode(t.generate()))
            else:
                Path(source).replace(torrent_path)

        files = ti.files()
        file_list = [{"path": files.file_path(i), "size": files.file_size(i)} for i in range(files.num_files())]
        metadata = {
            "name": ti.name(),
            "size_bytes": ti.total_size(),
            "num_files": ti.num_files(),
            "files": file_list,
            "num_pieces": ti.num_pieces(),
            "piece_length": ti.piece_length(),
            "info_hash": info_hash,
        }
        await send_success(writer, metadata)
    except Exception as e:
        await send_error(writer, str(e))
        