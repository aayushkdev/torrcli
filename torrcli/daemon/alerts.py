import asyncio
import libtorrent as lt
from pathlib import Path
from torrcli.daemon.session import ses, save_all_resume_data, on_save_resume_data
from torrcli.daemon.config import SOCKET_PATH

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
            if isinstance(alert, lt.save_resume_data_alert):
                on_save_resume_data(alert)
                pending_resume -= 1

        if shutdown_event.is_set():
            if pending_resume <= 0:
                print("All resume data saved. Shutting down.")
                Path(SOCKET_PATH).unlink(missing_ok=True)
                return
        else:
            pending_resume = len(ses.get_torrents())

        await asyncio.sleep(1)