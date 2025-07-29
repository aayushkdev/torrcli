#!/bin/bash
set -euo pipefail

echo "[+] Checking for required tools..."
for cmd in python3 sudo install systemctl; do
    command -v "$cmd" >/dev/null || { echo "$cmd is required but not found"; exit 1; }
done

if ! pidof systemd >/dev/null; then
    echo "[-] systemd not running. This script supports only systemd-based Linux systems."
    exit 1
fi

if ! command -v pipx >/dev/null; then
    echo "[!] pipx not found. Attempting to install..."

    if command -v pacman >/dev/null; then
        echo "[+] Installing pipx with pacman..."
        sudo pacman -Sy --noconfirm python-pipx
    elif command -v apt >/dev/null; then
        echo "[+] Installing pipx with apt..."
        sudo apt update
        sudo apt install -y pipx
        export PATH="$HOME/.local/bin:$PATH"
    elif command -v dnf >/dev/null; then
        echo "[+] Installing pipx with dnf..."
        sudo dnf install -y pipx
    else
        echo "❌ Unknown package manager. Please install pipx manually from https://pypa.github.io/pipx/"
        exit 1
    fi
fi

echo "[+] Installing torrcli via pipx..."
pipx install . --force

echo "[+] Copying config example to /usr/share/torrcli/..."
sudo install -Dm644 torrcli.conf.example /usr/share/torrcli/torrcli.conf.example

echo "[+] Preparing systemd service file..."
USER_NAME=${SUDO_USER:-$(whoami)}

TORRCLI_PYTHON="$HOME/.local/share/pipx/venvs/torrcli/bin/python"

sed "s|__USER__|$USER_NAME|; s|__PYTHON__|$TORRCLI_PYTHON|" torrcli.service > /tmp/torrcli.service.final

echo "[+] Installing systemd service..."
sudo install -Dm644 /tmp/torrcli.service.final /usr/lib/systemd/system/torrcli.service

echo "[+] Enabling and starting systemd service..."
sudo systemctl daemon-reexec
sudo systemctl daemon-reload
sudo systemctl enable torrcli.service
sudo systemctl start torrcli.service

echo "✅ Torrcli installed and running via systemd."