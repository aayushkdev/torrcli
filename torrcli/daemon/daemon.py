import asyncio
import signal
from torrcli.daemon.session import load_resume_and_torrents
from torrcli.daemon.alerts import clean_exit
from torrcli.daemon.ipc_server import socket_server

async def main_async():
    """Main async function that handles everything properly"""
    # Load torrents synchronously first
    load_resume_and_torrents()
    
    def signal_handler():
        print("Shutdown signal received...")
        clean_exit()
    
    # Set up signal handlers
    loop = asyncio.get_running_loop()
    loop.add_signal_handler(signal.SIGTERM, signal_handler)
    loop.add_signal_handler(signal.SIGINT, signal_handler)
    
    try:
        # Run the socket server
        await socket_server()
    except KeyboardInterrupt:
        print("Keyboard interrupt received")
    except Exception as e:
        print(f"Error in daemon: {e}")
    finally:
        print("Daemon shutting down...")

def main():
    try:
        asyncio.run(main_async())
    except KeyboardInterrupt:
        print("Daemon stopped")

if __name__ == "__main__":
    main()