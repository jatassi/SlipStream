@echo off
REM SlipStream Development Script
REM Run both backend and frontend in parallel

echo.
echo  SlipStream Development Environment
echo  ===================================
echo.

REM Start backend in a new window
start "SlipStream Backend :8080" cmd /k "cd /d %~dp0 && echo Starting Backend on http://localhost:8080 && echo. && go run ./cmd/slipstream"

REM Give backend a moment to start
timeout /t 2 /nobreak > nul

REM Start frontend in a new window
start "SlipStream Frontend :3000" cmd /k "cd /d %~dp0web && echo Starting Frontend on http://localhost:3000 && echo. && npm run dev"

echo  Development servers starting:
echo.
echo    Backend:  http://localhost:8080
echo    Frontend: http://localhost:3000
echo.
echo  Close the command windows to stop the servers.
echo.
