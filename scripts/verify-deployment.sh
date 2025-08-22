#!/bin/bash

# Bank REST API Deployment Verification Script
# This script verifies that the deployment is properly configured

set -e

echo "🔍 Bank REST API Deployment Verification"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
    else
        echo -e "${RED}✗${NC} $2"
    fi
}

# Function to print warning
print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

echo ""
echo "📋 Checking Prerequisites..."

# Check Docker
if command -v docker &> /dev/null; then
    DOCKER_VERSION=$(docker --version | cut -d' ' -f3 | cut -d',' -f1)
    print_status 0 "Docker installed (version $DOCKER_VERSION)"
else
    print_status 1 "Docker not found"
    exit 1
fi

# Check Docker Compose
if command -v docker-compose &> /dev/null; then
    COMPOSE_VERSION=$(docker-compose --version | cut -d' ' -f3 | cut -d',' -f1)
    print_status 0 "Docker Compose installed (version $COMPOSE_VERSION)"
else
    print_status 1 "Docker Compose not found"
    exit 1
fi

# Check Make
if command -v make &> /dev/null; then
    print_status 0 "Make utility available"
else
    print_status 1 "Make utility not found"
    print_warning "You can still use docker-compose commands directly"
fi

echo ""
echo "📁 Checking Project Structure..."

# Check required files
REQUIRED_FILES=(
    "Dockerfile"
    "docker-compose.yml"
    "docker-compose.prod.yml"
    "Makefile"
    ".env.example"
    ".gitignore"
    ".dockerignore"
    "go.mod"
    "go.sum"
    "cmd/server/main.go"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        print_status 0 "$file exists"
    else
        print_status 1 "$file missing"
    fi
done

# Check required directories
REQUIRED_DIRS=(
    "internal"
    "pkg"
    "docs"
    "test"
    "scripts"
    ".kiro/specs/bank-rest-api"
)

for dir in "${REQUIRED_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        print_status 0 "$dir/ directory exists"
    else
        print_status 1 "$dir/ directory missing"
    fi
done

echo ""
echo "🔒 Checking Security Configuration..."

# Check .env file
if [ -f ".env" ]; then
    print_status 0 ".env file exists"
    
    # Check for required environment variables
    REQUIRED_ENV_VARS=(
        "DB_PASSWORD"
        "REDIS_PASSWORD"
        "PASETO_SECRET_KEY"
        "SMTP_PASSWORD"
    )
    
    for var in "${REQUIRED_ENV_VARS[@]}"; do
        if grep -q "^${var}=" .env 2>/dev/null; then
            print_status 0 "$var is set in .env"
        else
            print_status 1 "$var not found in .env"
        fi
    done
else
    print_status 1 ".env file not found"
    print_warning "Run 'make dev-setup' or 'cp .env.example .env' to create it"
fi

# Check .gitignore for sensitive files
if grep -q "\.env" .gitignore; then
    print_status 0 ".env files excluded from git"
else
    print_status 1 ".env files not excluded from git"
fi

if grep -q "\*\.key" .gitignore; then
    print_status 0 "Key files excluded from git"
else
    print_status 1 "Key files not excluded from git"
fi

echo ""
echo "🐳 Checking Docker Configuration..."

# Check if Docker daemon is running
if docker info &> /dev/null; then
    print_status 0 "Docker daemon is running"
else
    print_status 1 "Docker daemon is not running"
    echo "Please start Docker and try again"
    exit 1
fi

# Check Docker Compose file syntax
if docker-compose config &> /dev/null; then
    print_status 0 "docker-compose.yml syntax is valid"
else
    print_status 1 "docker-compose.yml has syntax errors"
fi

if docker-compose -f docker-compose.prod.yml config &> /dev/null; then
    print_status 0 "docker-compose.prod.yml syntax is valid"
else
    print_status 1 "docker-compose.prod.yml has syntax errors"
fi

echo ""
echo "📚 Checking Documentation..."

DOC_FILES=(
    "README.md"
    "docs/API.md"
    "docs/DEPLOYMENT.md"
    "docs/TROUBLESHOOTING.md"
)

for doc in "${DOC_FILES[@]}"; do
    if [ -f "$doc" ]; then
        print_status 0 "$doc exists"
    else
        print_status 1 "$doc missing"
    fi
done

echo ""
echo "🧪 Testing Docker Build..."

# Test Docker build
if docker build -t bankapi-test . &> /dev/null; then
    print_status 0 "Docker image builds successfully"
    # Clean up test image
    docker rmi bankapi-test &> /dev/null
else
    print_status 1 "Docker build failed"
fi

echo ""
echo "📊 Summary"
echo "=========="

if [ -f ".env" ]; then
    echo -e "${GREEN}✓${NC} Ready for development deployment"
    echo ""
    echo "Next steps:"
    echo "1. Review and update .env file with your configuration"
    echo "2. Run 'make up' to start the development environment"
    echo "3. Visit http://localhost:8080/api/v1/health to verify deployment"
    echo ""
    echo "Available commands:"
    echo "  make up      - Start development environment"
    echo "  make logs    - View service logs"
    echo "  make health  - Check service health"
    echo "  make help    - Show all available commands"
else
    echo -e "${YELLOW}⚠${NC} Setup required"
    echo ""
    echo "Next steps:"
    echo "1. Run 'make dev-setup' to create .env file"
    echo "2. Edit .env file with your configuration"
    echo "3. Run 'make up' to start the development environment"
fi

echo ""
echo "📖 Documentation:"
echo "  README.md                 - Getting started guide"
echo "  docs/API.md              - Complete API reference"
echo "  docs/DEPLOYMENT.md       - Production deployment guide"
echo "  docs/TROUBLESHOOTING.md  - Common issues and solutions"

echo ""
echo "🎉 Verification complete!"