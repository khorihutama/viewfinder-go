$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$build = Join-Path $root 'build'
$dist = Join-Path $root 'dist\viewfinder-go'

New-Item -ItemType Directory -Force -Path $dist | Out-Null

$required = @(
    'viewfinder-go.exe',
    'viewfinder_native.dll',
    'fullscreen_vs.cso',
    'nv12_ps.cso'
)

foreach ($name in $required) {
    $source = Join-Path $build $name
    if (-not (Test-Path -LiteralPath $source)) {
        throw "Missing build artifact: $source"
    }
    Copy-Item -LiteralPath $source -Destination (Join-Path $dist $name) -Force
}

Copy-Item -LiteralPath (Join-Path $root 'README.md') -Destination (Join-Path $dist 'README.md') -Force
$revision = (git -C $root rev-parse --short HEAD 2>$null)
if (-not $revision) {
    throw 'Unable to resolve Git revision for release manifest'
}
$manifest = @{
    application = 'Viewfinder Go'
    revision = $revision
    packagedAt = (Get-Date).ToUniversalTime().ToString('o')
    artifacts = $required + @('README.md', 'release.json')
} | ConvertTo-Json
Set-Content -LiteralPath (Join-Path $dist 'release.json') -Value $manifest -Encoding utf8
Write-Output "Release package created: $dist"
