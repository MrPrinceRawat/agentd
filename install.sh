#!/bin/bash
set -e

REPO="MrPrinceRawat/agentd"
INSTALL_DIR="$HOME/.agentd/bin"
SERVICE_DIR="$HOME/.config/systemd/user"

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
if [ "$OS" != "linux" ]; then
    echo "agentd only supports Linux (got: $OS)"
    exit 1
fi

echo "Installing agentd (${OS}/${ARCH})..."

# Download binary
mkdir -p "$INSTALL_DIR"
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/agentd-${OS}-${ARCH}"
echo "Downloading from $DOWNLOAD_URL"
curl -sSL "$DOWNLOAD_URL" -o "$INSTALL_DIR/agentd"
chmod +x "$INSTALL_DIR/agentd"

echo "Binary installed to $INSTALL_DIR/agentd"

# Setup systemd user service if available
if command -v systemctl &>/dev/null && systemctl --user status &>/dev/null 2>&1; then
    mkdir -p "$SERVICE_DIR"

    # Determine socket path
    SOCK_PATH="/run/user/$(id -u)/agentd.sock"
    if [ ! -d "/run/user/$(id -u)" ]; then
        SOCK_PATH="$HOME/.agentd/agentd.sock"
    fi

    cat > "$SERVICE_DIR/agentd.service" << EOF
[Unit]
Description=agentd - persistent remote shell daemon
After=network.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/agentd --socket
Restart=on-failure
RestartSec=3

[Install]
WantedBy=default.target
EOF

    systemctl --user daemon-reload
    systemctl --user enable agentd
    systemctl --user start agentd

    # Enable linger so service runs without active login
    loginctl enable-linger "$(whoami)" 2>/dev/null || true

    echo "systemd service started and enabled"
    echo "Socket: $SOCK_PATH"

else
    # Fallback: nohup
    echo "systemd not available, starting with nohup..."
    mkdir -p "$HOME/.agentd"

    # Kill existing if running
    if [ -f "$HOME/.agentd/agentd.pid" ]; then
        kill "$(cat "$HOME/.agentd/agentd.pid")" 2>/dev/null || true
    fi

    nohup "$INSTALL_DIR/agentd" --socket > "$HOME/.agentd/agentd.log" 2>&1 &
    echo $! > "$HOME/.agentd/agentd.pid"

    echo "agentd started with nohup (PID: $!)"
    echo "Log: $HOME/.agentd/agentd.log"
fi

echo ""
echo "agentd installed successfully!"
