#!/bin/bash
set -e

echo "[+] Stopping and disabling systemd service..."
sudo systemctl stop torrcli.service || true
sudo systemctl disable torrcli.service || true

echo "[+] Removing systemd service file..."
sudo rm -f /usr/lib/systemd/system/torrcli.service
sudo systemctl daemon-reload

echo "[+] Removing installed package files..."

site_packages_dir=$(python3 -c "import site; print(site.getsitepackages()[0])")

sudo rm -rf $site_packages_dir/torrcli $site_packages_dir/torrcli-*

sudo rm -f /usr/bin/torrcli

sudo rm -rf "$site_packages_dir/torrcli-*.dist-info"

rm -f dist/*.whl

echo "✅ torrcli uninstalled!"