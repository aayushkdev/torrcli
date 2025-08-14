# Torrcli

A feature-rich, beautiful, and efficient terminal-based BitTorrent client built with Python, leveraging `rich` for stunning console UI and `libtorrent` for robust torrenting capabilities.

<img width="881" height="336" alt="Screenshot_20250814_213459" src="https://github.com/user-attachments/assets/81d5a1d3-6fd4-4292-8471-aff4d16519df" />



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

**pipx is required before installing so use apt/pacman/rpm to install pipx before continuing**

If you're not using Arch or prefer manual installation, follow these steps:

1. **Clone the Repository:**
```bash
git clone https://github.com/aayushkdev/torrcli.git
cd torrcli
```

2. **Run the Installer Script:**
```bash
./install.sh
```

3. **To Uninstall:**
```bash
./uninstall.sh
```

## ⚙️ Configuration

`torrcli` uses a configuration file to allow you to customize its behavior, such as download directories, port settings, and more.

The default configuration file is located at `~/.config/torrcli/config.conf`.

## 💡 Usage

Detailed commands:

- `torrcli add <source> [--save/-s SAVE_PATH] [--stream/-t]`
    Downloads a torrent from the specified source.

    - `<source>`: Can be a magnet link (e.g., `"magnet:?xt=urn:btih:..."`) or a path to a `.torrent` file (e.g., `/path/to/your/file.torrent`).

    - `--save/-s SAVE_PATH`: (Optional) Specify the directory where the downloaded files will be saved. If not provided, the `default_save_path` from your `config.conf` will be used.

    - `--stream/-t`: (Optional) Specify if the files will be downloaded sequentially

- `torrcli search <torrent-name> [--save/-s SAVE_PATH] [--stream/-t]`
    Searches for a torrent from various sources and lets you interactively select a torrent to download.
    It has all the same flags as `torrcli add`

- `torrcli ls`
    Lists all active torrents, showing their status, progress, and index.

- `torrcli info <index>`
    Shows detailed information about a specific torrent.

- `torrcli pause <index>`
    Pauses a running torrent.

- `torrcli resume <index>`
    Resumes a paused torrent.

- `torrcli rm <index>`
    Removes a torrent from the list. This does not delete the downloaded files by default.

```markdown
(Note: Replace `<index>` with the actual numerical index displayed by `torrcli list`)
```
