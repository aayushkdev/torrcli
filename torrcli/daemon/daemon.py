import asyncio
import signal
import sys
from torrcli.daemon.session import load_resume_and_torrents
from torrcli.daemon.alerts import shutdown_event
from torrcli.daemon.session import save_all_resume_data
from torrcli.daemon.ipc_server import socket_server
from torrcli.daemon.config import write_pid, remove_pid, check_pid

async def main_async():
    if check_pid():
        print("Daemon is already running. Exiting.")
        sys.exit(1)

    write_pid()

    await load_resume_and_torrents()

    shutdown_event.clear()
    save_all_resume_data_called = False

    def signal_handler():
        nonlocal save_all_resume_data_called
        if save_all_resume_data_called:
            return
        save_all_resume_data_called = True
        print("Shutdown signal received, saving resume data...")
        shutdown_event.set()
        save_all_resume_data()

    loop = asyncio.get_running_loop()
    loop.add_signal_handler(signal.SIGTERM, signal_handler)
    loop.add_signal_handler(signal.SIGINT, signal_handler)

    try:
        await socket_server()
    except Exception as e:
        print(f"Error in daemon: {e}")
        raise
    finally:
        print("Daemon shutting down...")
        remove_pid()

def main():
    try:
        asyncio.run(main_async())
    except KeyboardInterrupt:
        print("Daemon stopped")
    except SystemExit:
        raise
    except Exception as e:
        print(f"Fatal daemon error: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
