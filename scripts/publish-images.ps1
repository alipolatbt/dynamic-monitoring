param(
  [Parameter(Mandatory = $true)] [string] $Username,
  [Parameter(Mandatory = $false)] [string] $Version = "1.0.0"
)

# Requires prior 'docker login' to Docker Hub

$backendLocal = "dynamic_monitoring-backend:latest"
$frontendLocal = "dynamic_monitoring-frontend:latest"

$backendRemote = "$Username/dynamic-monitoring-backend:$Version"
$frontendRemote = "$Username/dynamic-monitoring-frontend:$Version"

Write-Host "Tagging images for Docker Hub..."
docker tag $backendLocal $backendRemote
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

docker tag $frontendLocal $frontendRemote
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "Pushing images to Docker Hub..."
docker push $backendRemote
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

docker push $frontendRemote
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

# Also push latest tags for convenience
$backendLatest = "$Username/dynamic-monitoring-backend:latest"
$frontendLatest = "$Username/dynamic-monitoring-frontend:latest"

docker tag $backendLocal $backendLatest
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

docker tag $frontendLocal $frontendLatest
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

docker push $backendLatest
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

docker push $frontendLatest
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host ""
Write-Host "✅ Successfully pushed images to Docker Hub!"
Write-Host ""
Write-Host "To use in production, create .env file:"
Write-Host "BACKEND_IMAGE=$backendRemote"
Write-Host "FRONTEND_IMAGE=$frontendRemote"
Write-Host ""
Write-Host "Then run: docker compose -f docker-compose.prod.yml up -d"
