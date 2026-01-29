#!/bin/bash
# Post-installation script for SlipStream

echo "SlipStream installed successfully."
echo ""
echo "To run SlipStream:"
echo "  slipstream"
echo ""
echo "On first run, SlipStream will set up its files in ~/.local/share/slipstream/"
echo "This enables automatic updates without requiring sudo."
echo ""
echo "To set up as a systemd user service (runs at login):"
echo "  mkdir -p ~/.config/systemd/user"
echo "  cp /usr/share/doc/slipstream/slipstream.service ~/.config/systemd/user/"
echo "  systemctl --user daemon-reload"
echo "  systemctl --user enable --now slipstream"
echo ""
echo "View logs: journalctl --user -u slipstream"
echo "Data stored in: ~/.local/share/slipstream/"
