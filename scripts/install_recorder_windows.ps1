$ErrorActionPreference = "Stop"

$repoUrl = "https://github.com/tm-LBenson/zoom_class_pipeline.git"
$binaryName = "zoom-recorder.exe"
$installDir = Join-Path $env:USERPROFILE "zoom-recorder"
$tempDir = Join-Path $env:TEMP "zoom_class_pipeline_install"

function Ensure-Tool {
    param(
        [Parameter(Mandatory=$true)][string]$Name
    )
    $cmd = Get-Command $Name -ErrorAction SilentlyContinue
    if (-not $cmd) {
        if (Get-Command winget -ErrorAction SilentlyContinue) {
            Write-Host "$Name is not installed." -ForegroundColor Yellow
            $answer = Read-Host "Press Y to install $Name with winget, or any other key to cancel"
            if ($answer -eq "Y" -or $answer -eq "y") {
                if ($Name -eq "git") {
                    winget install -e --id Git.Git
                } elseif ($Name -eq "go") {
                    winget install -e --id GoLang.Go
                }
            } else {
                Write-Host "$Name is required. Install it and run this script again." -ForegroundColor Red
                exit 1
            }
        } else {
            Write-Host "$Name is required. Install it and run this script again." -ForegroundColor Red
            exit 1
        }
    }
}

Ensure-Tool -Name "git"
Ensure-Tool -Name "go"

if (Test-Path $tempDir) {
    Remove-Item $tempDir -Recurse -Force
}
git clone --depth 1 $repoUrl $tempDir

Push-Location $tempDir
go build -o $binaryName
Pop-Location

if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

Move-Item -Force (Join-Path $tempDir $binaryName) (Join-Path $installDir $binaryName)

Remove-Item $tempDir -Recurse -Force

Write-Host ""
Write-Host "Installed $binaryName to $installDir" -ForegroundColor Green
Write-Host "Next steps:" -ForegroundColor Green
Write-Host "1) Open PowerShell"
Write-Host "2) cd `"$installDir`""
Write-Host "3) Run .\zoom-recorder.exe to generate config.json and then edit it."
