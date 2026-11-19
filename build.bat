@echo off
REM Builds Clash Go as a Windows GUI app and launches it.
REM Double-click this file to build and run; no PowerShell policy needed.
setlocal

cd /d "%~dp0"

echo Building clash-go.exe (GUI, no console window)...
go build -trimpath -ldflags "-H=windowsgui -s -w" -o clash-go.exe .
if errorlevel 1 (
  echo.
  echo Build FAILED.
  pause
  exit /b 1
)

echo.
echo Built clash-go.exe successfully.
echo.

REM Launch the app. Set SKIP_RUN=1 to build only.
if "%SKIP_RUN%"=="1" (
  echo SKIP_RUN=1, not launching.
) else (
  echo Starting clash-go.exe...
  start "" "%~dp0clash-go.exe"
)

echo.
pause
endlocal
