import asyncio
import libtorrent as lt
from pathlib import Path
from torrcli.daemon.session import ses, save_all_resume_data, on_save_resume_data, on_save_resume_failed, pending_resume_saves
from torrcli.daemon.config import SOCKET_PATH, SEED_AFTER_DOWNLOAD, REMOVE_AFTER_DOWNLOAD
from torrcli.daemon.commands.remove import remove_torrent

shutdown_event = asyncio.Event()

def clean_exit(*_):
    print("Shutdown requested, saving resume data...")
    shutdown_event.set()
    save_all_resume_data()

async def alert_loop():
    while not shutdown_event.is_set():
        alerts = ses.pop_alerts()
        
        if alerts:  # Only process if we have alerts
            for alert in alerts:
                try:
                    if isinstance(alert, lt.torrent_finished_alert):
                        handle = alert.handle
                        if REMOVE_AFTER_DOWNLOAD:
                            info_hash = str(handle.info_hashes().get_best())
                            await remove_torrent(info_hash)
                        elif not SEED_AFTER_DOWNLOAD:
                            handle.pause()

                    elif isinstance(alert, lt.save_resume_data_alert):
                        on_save_resume_data(alert)

                    elif isinstance(alert, lt.save_resume_data_failed_alert):
                        on_save_resume_failed(alert)
                        
                except Exception as e:
                    print(f"Error processing alert: {e}")

        # Check if we need to exit after saving resume data
        if shutdown_event.is_set():
            if all(pending_resume_saves.values()):
                print("All resume data saved. Exiting.")
                Path(SOCKET_PATH).unlink(missing_ok=True)
                return

        # Use shorter sleep when we have work, longer when idle
        sleep_time = 0.1 if alerts else 1.0
        await asyncio.sleep(sleep_time)