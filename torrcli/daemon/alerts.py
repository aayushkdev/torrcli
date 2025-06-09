import asyncio
import libtorrent as lt
from pathlib import Path
from torrcli.daemon.session import ses, save_all_resume_data, on_save_resume_data
from torrcli.daemon.config import SOCKET_PATH, SEED_AFTER_DOWNLOAD, REMOVE_AFTER_DOWNLOAD
from torrcli.daemon.commands.remove import remove_torrent

shutdown_event = asyncio.Event()

def clean_exit(*_):
    print("Shutdown requested. Saving resume data...")
    shutdown_event.set()
    save_all_resume_data()

async def alert_loop():
    pending_resume = 0
    while True:
        alerts = ses.pop_alerts()
        for alert in alerts:
            if isinstance(alert, lt.torrent_finished_alert):
                if REMOVE_AFTER_DOWNLOAD:
                    info_hash = str(alert.handle.info_hashes().get_best())
                    await remove_torrent(info_hash)

                if not SEED_AFTER_DOWNLOAD:
                    handle = alert.handle
                    handle.pause()

        if shutdown_event.is_set():
            if pending_resume <= 0:
                print("All resume data saved. Shutting down.")
                Path(SOCKET_PATH).unlink(missing_ok=True)
                return
        else:
            pending_resume = len(ses.get_torrents())

        await asyncio.sleep(1)