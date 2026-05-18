param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$ScriptName,

    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ScriptArgs
)

$ErrorActionPreference = "Stop"

$scriptDir = $PSScriptRoot
if (-not $scriptDir) {
    $scriptDir = (Get-Location).Path
}

$scriptPath = Join-Path $scriptDir $ScriptName
if (-not (Test-Path -LiteralPath $scriptPath -PathType Leaf)) {
    throw "Script not found: $scriptPath"
}

if ($scriptDir -match '^([A-Za-z]):\\(.*)$') {
    $drive = $matches[1].ToLowerInvariant()
    $rest = ($matches[2] -replace '\\', '/')
    $wslScriptDir = "/mnt/$drive/$rest"
} else {
    $wslScriptDir = ($scriptDir -replace '\\', '/')
}

if (-not $wslScriptDir) {
    throw "Failed to resolve WSL path for: $scriptDir"
}

Write-Host "Running ./$ScriptName in WSL from $wslScriptDir"
& wsl.exe --cd "$wslScriptDir" bash "./$ScriptName" @ScriptArgs
$exitCode = $LASTEXITCODE

if ($exitCode -ne 0) {
    Write-Error "$ScriptName failed with exit code $exitCode"
    exit $exitCode
}
