#!/bin/bash
set -euo pipefail

echo "[+] Stopping and disabling systemd service..."
sudo systemctl stop torrcli.service || true
sudo systemctl disable torrcli.service || true

echo "[+] Removing systemd service file..."
sudo rm -f /usr/lib/systemd/system/torrcli.service
sudo systemctl daemon-reload

echo "[+] Locating site-packages directory..."
site_packages_dir=$(python3 -c "import site; print(site.getsitepackages()[0])")

echo "[+] Removing Python package files..."
sudo rm -rf "$site_packages_dir/torrcli"
sudo rm -rf "$site_packages_dir/torrcli-"*.dist-info
sudo rm -f /usr/bin/torrcli 2>/dev/null || true

echo "[+] Removing shared config..."
sudo rm -rf /usr/share/torrcli/

echo "[+] Removing user config and cache..."
rm -rf ~/.local/torrcli/
rm -rf ~/.config/torrcli/

echo "[+] Cleaning build artifacts..."
rm -rf dist build *.egg-info

echo "torrcli completely uninstalled."