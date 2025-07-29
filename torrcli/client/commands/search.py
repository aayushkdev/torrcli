from rich.console import Console
from rich.prompt import Prompt
from torrfetch.core import search_torrents_async
from torrcli.client.ipc import send_and_receive
from torrcli.client.commands.progress import progress
from torrcli.client.commands.download import download
from torrcli.client.ui import show_metadata, show_torrent_search_results

console = Console()

async def search_and_download(query, save_path, stream=False):
    results = await search_torrents_async(query, mode="fallback")
    if not results:
        console.print(f"[bold red]No results found for '{query}'[/bold red]")
        return

    show_torrent_search_results(results, query)

    num_results = len(results)
    choice = Prompt.ask(
        f"Select a torrent to download [1–{num_results}]",
        choices=[str(i) for i in range(1, num_results + 1)],
        show_choices=False
    )

    selected = results[int(choice) - 1]

    await download(selected["magnet"], save_path, stream)