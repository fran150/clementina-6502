#!/bin/bash
# Setup script for Clementina 6502 on macOS
# This script removes the quarantine attribute that causes security warnings

# Get the directory where this script is located
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Remove quarantine attribute from the executable
echo "Removing quarantine attribute from Clementina 6502..."
xattr -d com.apple.quarantine "$DIR/clementina" 2>/dev/null || true

# Make sure the executable has proper permissions
chmod +x "$DIR/clementina"

echo "✅ Clementina 6502 is now ready to run!"
echo "   You can start it by running: ./clementina"
