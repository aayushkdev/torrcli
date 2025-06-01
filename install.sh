#!/bin/bash
set -euo pipefail

echo "[+] Checking for required tools..."
for cmd in python3 sudo install systemctl; do
    command -v $cmd >/dev/null || { echo "$cmd is required but not found"; exit 1; }
done

if ! pidof systemd >/dev/null; then
    echo "systemd not running. This script supports only systemd-based Linux systems."
    exit 1
fi

echo "[+] Building wheel..."
python3 -m build --wheel --no-isolation

echo "[+] Installing wheel system-wide..."
WHEEL_PATH=$(ls dist/*.whl | head -n 1)
sudo python3 -m installer --destdir=/ "$WHEEL_PATH" || true

echo "[+] Copying config example to /usr/share/torrcli/..."
sudo install -Dm644 torrcli.conf.example /usr/share/torrcli/torrcli.conf.example

echo "[+] Installing systemd service..."
sudo install -Dm644 torrcli.service /usr/lib/systemd/system/torrcli.service

USER_NAME=${SUDO_USER:-$(whoami)}
sudo sed -i "s/__USER__/$USER_NAME/" /usr/lib/systemd/system/torrcli.service

echo "[+] Enabling and starting systemd service..."
sudo systemctl daemon-reexec
sudo systemctl enable torrcli.service
sudo systemctl start torrcli.service

echo "torrcli installed and service running!"