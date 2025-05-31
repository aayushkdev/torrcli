#!/bin/bash
set -e

echo "[+] Building wheel..."
python3 -m build --wheel --no-isolation

echo "[+] Installing wheel system-wide..."
sudo python3 -m installer --destdir=/ dist/*.whl

echo "[+] Installing systemd service..."
sudo install -Dm644 torrcli.service /usr/lib/systemd/system/torrcli.service

echo "[+] Enabling and starting systemd service..."
sudo systemctl daemon-reexec
sudo systemctl enable torrcli.service
sudo systemctl start torrcli.service

echo "✅ torrcli installed and service running!"