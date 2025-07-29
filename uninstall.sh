#!/bin/bash
set -euo pipefail

echo "[+] Stopping and disabling systemd service..."
sudo systemctl stop torrcli.service 2>/dev/null || true
sudo systemctl disable torrcli.service 2>/dev/null || true
sudo rm -f /usr/lib/systemd/system/torrcli.service
sudo systemctl daemon-reexec

echo "[+] Removing shared config..."
sudo rm -rf /usr/share/torrcli/

echo "[+] Uninstalling torrcli pipx environment..."
pipx uninstall torrcli

echo "[+] Removing user config and cache..."
rm -rf ~/.config/torrcli/ ~/.local/torrcli/ ~/.cache/torrcli/

echo "torrcli fully uninstalled"
