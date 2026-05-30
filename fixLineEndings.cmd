@echo off
setlocal EnableExtensions EnableDelayedExpansion

set "ROOT=%~dp0"
if "%ROOT:~-1%"=="\" set "ROOT=%ROOT:~0,-1%"
set "TMP_LIST=%TEMP%\rdos-lf-files-%RANDOM%%RANDOM%.txt"

if /I "%~1"=="-h" goto :help
if /I "%~1"=="--help" goto :help

where git >nul 2>&1
if errorlevel 1 (
    echo ERROR: git is required but was not found in PATH.
    exit /b 1
)

git -C "%ROOT%" rev-parse --is-inside-work-tree >nul 2>&1
if errorlevel 1 (
    echo ERROR: %ROOT% is not a Git working tree.
    exit /b 1
)

for /f %%I in ('git -C "%ROOT%" status --porcelain') do (
    echo ERROR: Working tree is not clean.
    echo Commit or stash changes first, then run fixLineEndings.cmd again.
    exit /b 1
)

echo Normalizing tracked files to LF based on .gitattributes...
git -C "%ROOT%" add --renormalize .
if errorlevel 1 (
    echo ERROR: git add --renormalize failed.
    exit /b 1
)

git -C "%ROOT%" diff --cached --name-only --diff-filter=ACMRT > "%TMP_LIST%"
if errorlevel 1 (
    echo ERROR: Could not read renormalized file list.
    del "%TMP_LIST%" >nul 2>&1
    exit /b 1
)

set "CHANGED=0"
for /f "usebackq delims=" %%F in ("%TMP_LIST%") do (
    set /a CHANGED+=1 >nul
    git -C "%ROOT%" checkout-index -f -- "%%F" >nul 2>&1
)

git -C "%ROOT%" restore --staged . >nul 2>&1
if errorlevel 1 (
    git -C "%ROOT%" reset --quiet HEAD -- . >nul 2>&1
)

del "%TMP_LIST%" >nul 2>&1

if "%CHANGED%"=="0" (
    echo No LF normalization changes were needed.
    exit /b 0
)

echo Done. Normalized %CHANGED% file^(s^) to LF in your working tree.
echo Review with: git status --short
exit /b 0

:help
echo fixLineEndings.cmd
echo.
echo Normalizes tracked files to LF using your repo .gitattributes rules.
echo.
echo Usage:
echo   fixLineEndings.cmd
echo.
echo Notes:
echo   - Requires a clean working tree.
echo   - Updates files in place and leaves changes unstaged.
exit /b 0
