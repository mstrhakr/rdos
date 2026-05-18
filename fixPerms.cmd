@echo off
setlocal enabledelayedexpansion

set "ROOT=%~dp0"
set "found=0"

echo Applying scoped ACLs to build artifacts in "%ROOT%"

for %%F in ("%ROOT%*.vhd" "%ROOT%*.iso") do (
    if exist "%%~fF" (
        set "found=1"
        echo   - %%~nxF
        icacls "%%~fF" /inheritance:e /grant:r "%USERNAME%":F "BUILTIN\Administrators":F "NT AUTHORITY\SYSTEM":F >nul
        if errorlevel 1 (
            echo     ! Failed to apply scoped ACLs on %%~nxF
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
