#!/bin/bash
# Post-installation script for SlipStream

# Create data directory
mkdir -p /var/lib/slipstream/data
chmod 755 /var/lib/slipstream

# Create config directory and blank config.yaml if it doesn't exist
mkdir -p /etc/slipstream
if [ ! -f /etc/slipstream/config.yaml ]; then
    touch /etc/slipstream/config.yaml
    chmod 644 /etc/slipstream/config.yaml
fi

# Create systemd service file if systemd is available
if command -v systemctl &> /dev/null; then
    cat > /etc/systemd/system/slipstream.service << 'EOF'
[Unit]
Description=SlipStream Media Management
After=network.target

[Service]
Type=simple
ExecStart=/usr/bin/slipstream --config /etc/slipstream/config.yaml
WorkingDirectory=/var/lib/slipstream
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
fi

echo "SlipStream installed successfully."
echo "Start with: systemctl start slipstream"
echo "Or run directly: slipstream"
