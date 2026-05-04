$ErrorActionPreference = "Stop"

$cmdArgs = @()
if ($args.Count -eq 0) {
  $cmdArgs = @("release", "--snapshot", "--clean")
} else {
  $cmdArgs = $args
}

$goreleaser = Get-Command goreleaser -ErrorAction SilentlyContinue
if ($null -ne $goreleaser) {
  & $goreleaser.Source @cmdArgs
  exit $LASTEXITCODE
}

$go = Get-Command go -ErrorAction SilentlyContinue
if ($null -eq $go -and (Test-Path "C:\Program Files\Go\bin\go.exe")) {
  $go = @{ Source = "C:\Program Files\Go\bin\go.exe" }
}

if ($null -eq $go) {
  throw "Neither 'goreleaser' nor 'go' was found. Install Go or Goreleaser, then try again."
}

& $go.Source run github.com/goreleaser/goreleaser/v2@latest @cmdArgs
exit $LASTEXITCODE
