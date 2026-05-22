@echo off
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0Invoke-BashScript.ps1" setup-build-sudoers.sh %*
set "exitcode=%ERRORLEVEL%"
exit /b %exitcode%
