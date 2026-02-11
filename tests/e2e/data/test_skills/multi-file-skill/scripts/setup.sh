#!/bin/bash

# Setup script for multi-file-skill
# This script sets up the environment and installs dependencies

set -e

echo "Setting up multi-file-skill..."

# Check Python version
PYTHON_VERSION=$(python3 --version 2>&1 | awk '{print $2}')
echo "Python version: $PYTHON_VERSION"

# Check if Python 3.8 or higher
if [[ $(echo "$PYTHON_VERSION 3.8" | awk '{print ($1 >= $2)}') -eq 0 ]]; then
    echo "Error: Python 3.8 or higher is required"
    exit 1
fi

# Create virtual environment
if [ ! -d "venv" ]; then
    echo "Creating virtual environment..."
    python3 -m venv venv
fi

# Activate virtual environment
source venv/bin/activate

# Upgrade pip
echo "Upgrading pip..."
pip install --upgrade pip

# Install dependencies
echo "Installing dependencies..."
pip install requests>=2.25.0

# Create necessary directories
echo "Creating directories..."
mkdir -p logs
mkdir -p data

# Set permissions
echo "Setting permissions..."
chmod +x scripts/*.sh 2>/dev/null || true

# Create .env file if it doesn't exist
if [ ! -f ".env" ]; then
    echo "Creating .env file..."
    cat > .env << EOF
# Environment variables for multi-file-skill
API_TOKEN=your_token_here
LOG_LEVEL=INFO
DEBUG=false
EOF
    echo "Created .env file. Please update with your actual values."
fi

echo "Setup completed successfully!"
echo ""
echo "Next steps:"
echo "1. Update .env file with your configuration"
echo "2. Activate virtual environment: source venv/bin/activate"
echo "3. Run tests: python -m pytest tests/"
echo "4. Use the skill with: skill-hub use multi-file-skill"