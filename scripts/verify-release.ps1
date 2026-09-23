$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root 'dist\viewfinder-go'
$manifestPath = Join-Path $dist 'release.json'

if (-not (Test-Path -LiteralPath $manifestPath)) {
    throw "Missing release manifest: $manifestPath"
}

$manifest = Get-Content -Raw -LiteralPath $manifestPath | ConvertFrom-Json
foreach ($artifact in $manifest.artifacts) {
    $path = Join-Path $dist $artifact
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Missing release artifact: $artifact"
    }
}

if ([string]::IsNullOrWhiteSpace($manifest.revision)) {
    throw 'Release revision is empty'
}

Write-Output "Release verified: $($manifest.application) revision $($manifest.revision)"
