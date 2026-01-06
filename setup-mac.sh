#!/bin/bash

# SlipStream macOS Setup Script
# This script configures a complete development environment for SlipStream on macOS

set -e  # Exit on any error

echo "🚀 Setting up SlipStream development environment..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
print_status() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if running on macOS
if [[ "$OSTYPE" != "darwin"* ]]; then
    print_error "This script is designed for macOS only"
    exit 1
fi

# Check if Homebrew is installed
if ! command -v brew &> /dev/null; then
    print_status "Installing Homebrew..."
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    print_success "Homebrew installed"
else
    print_success "Homebrew already installed"
fi

# Function to install a tool if not present
install_if_missing() {
    local tool=$1
    local install_cmd=$2
    local check_cmd=$3
    
    if ! command -v $tool &> /dev/null; then
        print_status "Installing $tool..."
        eval $install_cmd
        print_success "$tool installed"
    else
        print_success "$tool already installed"
        if [ -n "$check_cmd" ]; then
            eval $check_cmd
        fi
    fi
}

# Install required tools
print_status "Checking and installing required tools..."

# Go
install_if_missing "go" "brew install go" "go version"

# Node.js
install_if_missing "node" "brew install node" "node --version"

# npm (comes with Node.js)
if ! command -v npm &> /dev/null; then
    print_error "npm not found even though Node.js is installed"
    exit 1
fi

# Make (usually pre-installed on macOS)
if ! command -v make &> /dev/null; then
    print_status "Installing make..."
    brew install make
    print_success "make installed"
else
    print_success "make already installed"
fi

# Git
install_if_missing "git" "brew install git" "git --version"

# Install sqlc (SQL code generation tool)
if ! command -v sqlc &> /dev/null; then
    print_status "Installing sqlc..."
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
    
    # Add to PATH if not already there
    if ! echo $PATH | grep -q "$HOME/go/bin"; then
        echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
        export PATH="$HOME/go/bin:$PATH"
        print_warning "Added ~/go/bin to PATH in ~/.zshrc. Please restart your terminal or run: source ~/.zshrc"
    fi
    
    # Verify installation
    if ~/go/bin/sqlc version &> /dev/null; then
        print_success "sqlc installed"
    else
        print_error "sqlc installation failed"
        exit 1
    fi
else
    print_success "sqlc already installed"
    sqlc version
fi

# Navigate to project directory (assuming script is run from project root)
if [ ! -f "go.mod" ] || [ ! -f "package.json" ]; then
    print_error "Please run this script from the SlipStream project root directory"
    exit 1
fi

print_status "Setting up project configuration..."

# Copy configuration files if they don't exist
if [ ! -f "config.yaml" ]; then
    cp configs/config.example.yaml config.yaml
    print_success "Created config.yaml from template"
else
    print_warning "config.yaml already exists, skipping"
fi

if [ ! -f ".env" ]; then
    cp .env.example .env
    print_success "Created .env from template"
else
    print_warning ".env already exists, skipping"
fi

# Install project dependencies
print_status "Installing Go dependencies..."
go mod download
print_success "Go dependencies installed"

print_status "Installing frontend dependencies..."
cd web
npm install
cd ..
print_success "Frontend dependencies installed"

# Create necessary directories
print_status "Creating necessary directories..."
mkdir -p data bin
print_success "Directories created"

# Generate SQL code
print_status "Generating SQL code..."
sqlc generate
print_success "SQL code generated"

# Check if API keys are configured
print_status "Checking API key configuration..."

if grep -q "your_tmdb_api_key" .env 2>/dev/null || grep -q "your_tvdb_api_key" .env 2>/dev/null; then
    print_warning "⚠️  API keys not configured in .env file"
    echo ""
    echo "To complete setup, you'll need to:"
    echo "1. Get a TMDB API key: https://www.themoviedb.org/settings/api"
    echo "2. Get a TVDB API key: https://thetvdb.com/api-information"
    echo "3. Edit .env file and replace the placeholder keys"
    echo ""
    echo "Without API keys, metadata fetching will not work."
else
    print_success "API keys appear to be configured"
fi

# Test the setup
print_status "Testing setup..."

# Test Go build
if go build -o bin/slipstream ./cmd/slipstream; then
    print_success "Go build test passed"
    rm -f bin/slipstream  # Clean up test build
else
    print_error "Go build test failed"
    exit 1
fi

# Test frontend build
cd web
if npm run build; then
    print_success "Frontend build test passed"
else
    print_error "Frontend build test failed"
    exit 1
fi
cd ..

print_success "🎉 Setup completed successfully!"
echo ""
echo "Next steps:"
echo "1. Configure API keys in .env file (if not done already)"
echo "2. Run 'make dev' to start development servers"
echo "3. Backend will be available at http://localhost:8080"
echo "4. Frontend will be available at http://localhost:3000"
echo ""
echo "Useful commands:"
echo "- make dev              # Run both backend and frontend"
echo "- make dev-backend      # Run backend only"
echo "- make dev-frontend     # Run frontend only"
echo "- make test             # Run Go tests"
echo "- make build            # Build for production"
echo "- sqlc generate         # Regenerate SQL code after database changes"
echo ""
echo "If sqlc command is not found, restart your terminal or run: source ~/.zshrc"