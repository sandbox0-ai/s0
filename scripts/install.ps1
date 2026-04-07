$ErrorActionPreference = "Stop"

$Repo = "sandbox0-ai/s0"

function Resolve-Version {
  if ($env:S0_VERSION) {
    return $env:S0_VERSION
  }

  $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
  if (-not $release.tag_name) {
    throw "failed to resolve latest s0 release version"
  }
  return $release.tag_name
}

function Resolve-Arch {
  switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { return "amd64" }
    "ARM64" { return "arm64" }
    default { throw "unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
  }
}

$Version = Resolve-Version
$Arch = Resolve-Arch
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { Join-Path $HOME ".local\bin" }
$Archive = "s0-windows-$Arch.zip"
$Url = "https://github.com/$Repo/releases/download/$Version/$Archive"
$BinaryName = "s0-windows-$Arch.exe"

$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("s0-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $TempDir | Out-Null

try {
  New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

  $ZipPath = Join-Path $TempDir $Archive
  Invoke-WebRequest -Uri $Url -OutFile $ZipPath
  Expand-Archive -Path $ZipPath -DestinationPath $TempDir -Force

  $SourceBinary = Join-Path $TempDir $BinaryName
  if (-not (Test-Path $SourceBinary)) {
    $SourceBinary = Join-Path $TempDir "s0.exe"
  }

  Copy-Item -Path $SourceBinary -Destination (Join-Path $InstallDir "s0.exe") -Force
  Write-Host "installed s0 to $(Join-Path $InstallDir 's0.exe')"

  $currentUserPath = [Environment]::GetEnvironmentVariable("Path", "User")
  $normalizedPath = @($currentUserPath -split ';') | ForEach-Object { $_.TrimEnd('\') }
  if ($normalizedPath -notcontains $InstallDir.TrimEnd('\')) {
    Write-Warning "$InstallDir is not on PATH"
  }
}
finally {
  if (Test-Path $TempDir) {
    Remove-Item -Recurse -Force $TempDir
  }
}
