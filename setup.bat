@echo off
REM SlipStream Development Environment Setup Script
REM Run this script after cloning the repo to set up your dev environment

setlocal EnableDelayedExpansion

echo.
echo  SlipStream Development Environment Setup
echo  =========================================
echo.

REM Check if Go is installed
where go >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo  [ERROR] Go is not installed or not in PATH.
    echo          Please install Go from https://go.dev/dl/
    echo.
    exit /b 1
)

REM Check if Node.js is installed
where node >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo  [ERROR] Node.js is not installed or not in PATH.
    echo          Please install Node.js from https://nodejs.org/
    echo.
    exit /b 1
)

REM Check if npm is installed
where npm >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo  [ERROR] npm is not installed or not in PATH.
    echo          npm should come with Node.js installation.
    echo.
    exit /b 1
)

echo  [OK] Go version:
go version
echo.

echo  [OK] Node version:
node --version
echo.

echo  [OK] npm version:
call npm --version
echo.

REM Install Go dependencies
echo.
echo  [1/4] Installing Go dependencies...
echo.
go mod download
if %ERRORLEVEL% neq 0 (
    echo  [ERROR] Failed to install Go dependencies.
    exit /b 1
)
echo  [OK] Go dependencies installed.

REM Install sqlc (optional but recommended)
echo.
echo  [2/4] Installing sqlc (database query generator)...
echo.
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
if %ERRORLEVEL% neq 0 (
    echo  [WARN] Failed to install sqlc. This is optional but recommended.
    echo         You can install it manually later: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
) else (
    echo  [OK] sqlc installed.
)

REM Install root npm dependencies
echo.
echo  [3/4] Installing root npm dependencies...
echo.
call npm install
if %ERRORLEVEL% neq 0 (
    echo  [ERROR] Failed to install root npm dependencies.
    exit /b 1
)
echo  [OK] Root npm dependencies installed.

REM Install frontend npm dependencies
echo.
echo  [4/4] Installing frontend npm dependencies...
echo.
cd web
call npm install
if %ERRORLEVEL% neq 0 (
    echo  [ERROR] Failed to install frontend npm dependencies.
    exit /b 1
)
cd ..
echo  [OK] Frontend npm dependencies installed.

REM Create configs directory if it doesn't exist
if not exist "configs" (
    mkdir configs
    echo  [OK] Created configs directory.
)

REM Check for .env file
if not exist "configs\.env" (
    if exist "configs\.env.example" (
        copy "configs\.env.example" "configs\.env" >nul
        echo  [OK] Created configs\.env from example.
    ) else (
        echo  [INFO] No .env file found. You may need to create one for API keys.
    )
)

echo.
echo  =========================================
echo  Setup Complete!
echo  =========================================
echo.
echo  To start the development servers, run:
echo.
echo      dev.bat
echo.
echo  This will start:
echo    - Backend on http://localhost:8080
echo    - Frontend on http://localhost:3000
echo.
echo  Other useful commands:
echo    - go test ./...              Run Go tests
echo    - sqlc generate              Regenerate database code
echo    - cd web ^&^& npm run build    Build frontend for production
echo.

endlocal
