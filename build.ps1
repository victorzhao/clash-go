# Builds Clash Go as a Windows GUI app, so launching it opens no console
# window. Run: powershell -ExecutionPolicy Bypass -File build.ps1
$ErrorActionPreference = "Stop"
go build -trimpath -ldflags "-H=windowsgui -s -w" -o clash-go.exe .
Write-Host "built clash-go.exe (no console)"
