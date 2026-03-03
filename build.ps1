$ErrorActionPreference = "Stop"

$env:GOOS="linux"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"

Write-Host "Compiling Go binary for Amazon Linux..."
go build -tags lambda.norpc -o bootstrap .
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "Zipping..."
if (Test-Path lambda.zip) { Remove-Item lambda.zip }
Compress-Archive -Path bootstrap -DestinationPath lambda.zip

Write-Host "Cleaning up bootstrap file..."
Remove-Item bootstrap

Write-Host "Done! Use 'sam deploy --guided' to upload lambda.zip."
