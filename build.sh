#!/bin/sh
# Builds Clash Go. On Windows the binary is a GUI app, so launching it opens
# no console window.
set -e
if [ "$(go env GOOS)" = "windows" ]; then
	go build -trimpath -ldflags "-H=windowsgui -s -w" -o clash-go.exe .
	echo "built clash-go.exe (no console)"
else
	go build -trimpath -ldflags "-s -w" -o clash-go .
	echo "built clash-go"
fi
