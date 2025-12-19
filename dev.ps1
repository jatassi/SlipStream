# SlipStream Development Script
# Run both backend and frontend in parallel

Write-Host "Starting SlipStream Development Environment..." -ForegroundColor Cyan

# Start backend in a new PowerShell window
$backendJob = Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; Write-Host 'Starting Backend on :8080...' -ForegroundColor Green; go run ./cmd/slipstream" -PassThru

# Give backend a moment to start
Start-Sleep -Seconds 2

# Start frontend in a new PowerShell window
$frontendJob = Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD\web'; Write-Host 'Starting Frontend on :3000...' -ForegroundColor Green; npm run dev" -PassThru

Write-Host ""
Write-Host "Development servers starting:" -ForegroundColor Yellow
Write-Host "  Backend:  http://localhost:8080" -ForegroundColor White
Write-Host "  Frontend: http://localhost:3000" -ForegroundColor White
Write-Host ""
Write-Host "Close the PowerShell windows to stop the servers." -ForegroundColor Gray
