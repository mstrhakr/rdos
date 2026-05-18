@echo off
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0Invoke-BashScript.ps1" build.sh %*
set "exitcode=%ERRORLEVEL%"
exit /b %exitcode%
