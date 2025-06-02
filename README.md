# Torrcli

A feature-rich, beautiful, and efficient terminal-based BitTorrent client built with Python, leveraging `rich` for stunning console UI and `libtorrent` for robust torrenting capabilities.

## 🌟 Features

- **Intuitive Command-Line Interface**: Navigate and manage your torrents directly from your terminal.
- **Beautiful Console Output**: Powered by `rich`, enjoy vibrant colors, progress bars, and structured information.
- **Robust Torrent Engine**: Utilizes `libtorrent` for reliable and high-performance torrent downloads and uploads.
- **Lightweight & Fast**: Designed for minimal resource consumption, ideal for servers and low-power devices.
- **Configurable**: Customize torrcli behavior via a simple configuration file.
- **Cross-Platform (Python compatible)**: Run torrcli on any system where Python and its dependencies are supported.

## 🚀 Installation

torrcli offers flexible installation methods to suit your preference.

### Prerequisites

Before installing, ensure you have Python 3.8 or newer installed on your system.

### Option 1: Arch User Repository (AUR)

For Arch Linux users, torrcli is available in the AUR. This is the recommended method for easy installation and updates.

You can use any AUR helper like yay, paru, etc.
```bash
yay -S torrcli
```

### Option 2: Manual Install (Any Linux Distribution)

If you're not using Arch or prefer manual installation, follow these steps:

1. **Clone the Repository:**
```bash
git clone https://github.com/yourusername/torrcli.git
cd torrcli
```

2. **Install Python Dependencies:**
```bash
pip install rich
pip install libtorrent
```

3. **Run the Installer Script:**
```bash
./install.sh
```

4. **To Uninstall:**
```bash
./uninstall.sh
```

## ⚙️ Configuration

`torrcli` uses a configuration file to allow you to customize its behavior, such as download directories, port settings, and more.

The default configuration file is located at `~/.config/torrcli/config.conf`.

## 💡 Usage

Detailed commands:

- `torrcli download <source> [--save SAVE_PATH]`
    Download a torrent from the specified source.

    - `<source>`: Can be a magnet link (e.g., `"magnet:?xt=urn:btih:..."`) or a path to a `.torrent` file (e.g., `/path/to/your/file.torrent`).

    - `--save SAVE_PATH`: (Optional) Specify the directory where the downloaded files will be saved. If not provided, the `default_save_path` from your `config.conf` will be used.

- `torrcli list`
    List all active torrents, showing their status, progress, and index.

- `torrcli info <index>`
    Show detailed information about a specific torrent.

- `torrcli pause <index>`
    Pause a running torrent.

- `torrcli resume <index>`
    Resume a paused torrent.

- `torrcli remove <index>`
    Remove a torrent from the list. This does not delete the downloaded files by default.

```markdown
(Note: Replace `<index>` with the actual numerical index displayed by `torrcli list`)
```