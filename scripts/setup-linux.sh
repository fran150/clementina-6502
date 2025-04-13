#!/bin/bash
# Setup script for Clementina 6502 on Linux
# This script ensures proper permissions for the executable

# Get the directory where this script is located
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Make sure the executable has proper permissions
echo "Setting executable permissions for Clementina 6502..."
chmod +x "$DIR/clementina"

echo "✅ Clementina 6502 is now ready to run!"
echo "   You can start it by running: ./clementina"
