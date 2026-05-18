@echo off
setlocal enabledelayedexpansion

set "ROOT=%~dp0"
set "found=0"

echo Applying permissive ACLs to build artifacts in "%ROOT%"

for %%F in ("%ROOT%*.vhd" "%ROOT%*.iso") do (
    if exist "%%~fF" (
        set "found=1"
        echo   - %%~nxF
        icacls "%%~fF" /grant *S-1-1-0:F >nul
        if errorlevel 1 (
            echo     ! Failed to grant Everyone full control on %%~nxF
        ) else (
            echo     OK
        )
    )
)

if "%found%"=="0" (
    echo No .vhd or .iso files found. Nothing to update.
)

endlocal
exit /b 0
