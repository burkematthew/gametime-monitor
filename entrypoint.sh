#!/bin/sh
set -e

echo "Running migrations..."
gametime-monitor migrate up

echo "Starting application..."
exec gametime-monitor serve
