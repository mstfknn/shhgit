#!/bin/bash

# shhgit Automated Deploy Script
# Pulls latest changes from GitHub and deploys the project
# Handles Docker and Docker Compose installation on first run

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Find script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}shhgit Automated Deploy Script${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Root check
if [ "$EUID" -eq 0 ]; then
   echo -e "${RED}Do not run this script as root. Use sudo when needed.${NC}"
   exit 1
fi

# Git check
if ! command -v git &> /dev/null; then
    echo -e "${YELLOW}Git not found. Installing...${NC}"
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        sudo apt update
        sudo apt install git -y
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        echo -e "${RED}Please install Git via Homebrew: brew install git${NC}"
        exit 1
    fi
fi

# Docker installation check
if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}Docker not found. Installing...${NC}"
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        curl -fsSL https://get.docker.com -o get-docker.sh
        sudo sh get-docker.sh
        sudo usermod -aG docker $USER
        rm get-docker.sh
        echo -e "${GREEN}Docker installed. Please log out and back in, then run this script again.${NC}"
        exit 0
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        echo -e "${RED}Please install Docker Desktop: https://www.docker.com/products/docker-desktop${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}Docker is already installed${NC}"
fi

# Docker Compose installation check
DOCKER_COMPOSE_CMD=""
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker-compose"
    echo -e "${GREEN}Docker Compose (docker-compose) is already installed${NC}"
elif docker compose version &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker compose"
    echo -e "${GREEN}Docker Compose (docker compose) is already installed${NC}"
else
    echo -e "${YELLOW}Docker Compose not found. Installing...${NC}"
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        sudo apt update
        sudo apt install docker-compose-plugin -y || sudo apt install docker-compose -y
        if command -v docker-compose &> /dev/null; then
            DOCKER_COMPOSE_CMD="docker-compose"
            echo -e "${GREEN}Docker Compose (docker-compose) installed${NC}"
        elif docker compose version &> /dev/null; then
            DOCKER_COMPOSE_CMD="docker compose"
            echo -e "${GREEN}Docker Compose (docker compose) installed${NC}"
        else
            echo -e "${RED}Docker Compose installation failed!${NC}"
            exit 1
        fi
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        echo -e "${RED}Docker Compose should come with Docker Desktop. Please restart Docker Desktop.${NC}"
        exit 1
    fi
fi

# Start Docker service (Linux)
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo -e "${BLUE}Checking Docker service...${NC}"
    sudo systemctl start docker 2>/dev/null || true
    sudo systemctl enable docker 2>/dev/null || true
fi

# Git repository check
if [ ! -d ".git" ]; then
    echo -e "${YELLOW}This directory is not a Git repository.${NC}"
    echo -e "${YELLOW}Skipping git pull...${NC}"
    SKIP_GIT_PULL=true
else
    SKIP_GIT_PULL=false
fi

# Git pull
if [ "$SKIP_GIT_PULL" = false ]; then
    echo -e "${BLUE}Pulling latest changes from GitHub...${NC}"

    CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
    echo -e "${BLUE}Current branch: ${CURRENT_BRANCH}${NC}"

    # Stash uncommitted changes if any
    if ! git diff-index --quiet HEAD -- 2>/dev/null; then
        echo -e "${YELLOW}Uncommitted changes found. Stashing...${NC}"
        git stash push -m "Auto-stash before deploy $(date +%Y-%m-%d_%H-%M-%S)" || true
    fi

    if git pull origin "$CURRENT_BRANCH" 2>/dev/null; then
        echo -e "${GREEN}Git pull successful${NC}"
    else
        echo -e "${YELLOW}Git pull failed or no remote configured. Continuing...${NC}"
    fi
fi

# Config file check
if [ ! -f "config.yaml" ]; then
    echo -e "${RED}config.yaml not found!${NC}"
    echo -e "${YELLOW}Please create config.yaml and add your GitHub tokens.${NC}"
    exit 1
fi

# .env file check
if [ ! -f ".env" ]; then
    echo -e "${YELLOW}WARNING: .env file not found!${NC}"
    echo -e "${YELLOW}Please create a .env file with your GitHub tokens:${NC}"
    echo -e "${YELLOW}  GITHUB_TOKEN_1=ghp_your_token_1${NC}"
    echo -e "${YELLOW}  GITHUB_TOKEN_2=ghp_your_token_2${NC}"
    read -p "Continue without .env? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Port check (optional)
PORT=8080
if command -v netstat &> /dev/null; then
    if sudo netstat -tuln 2>/dev/null | grep -q ":$PORT "; then
        echo -e "${YELLOW}WARNING: Port $PORT is already in use!${NC}"
        read -p "Continue anyway? (y/n) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
fi

# Firewall check (Linux)
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    if command -v ufw &> /dev/null; then
        if sudo ufw status 2>/dev/null | grep -q "Status: active"; then
            if ! sudo ufw status 2>/dev/null | grep -q "$PORT/tcp"; then
                echo -e "${BLUE}Opening port $PORT in firewall...${NC}"
                sudo ufw allow $PORT/tcp 2>/dev/null || true
            fi
        fi
    fi
fi

# Stop containers
echo ""
echo -e "${BLUE}Stopping existing containers...${NC}"
$DOCKER_COMPOSE_CMD down 2>/dev/null || true

# Build Docker images
echo ""
echo -e "${BLUE}Building Docker images...${NC}"
if $DOCKER_COMPOSE_CMD build; then
    echo -e "${GREEN}Build successful${NC}"
else
    echo -e "${RED}Build failed!${NC}"
    exit 1
fi

# Start containers
echo ""
echo -e "${BLUE}Starting containers...${NC}"
if $DOCKER_COMPOSE_CMD up -d; then
    echo -e "${GREEN}Containers started${NC}"
else
    echo -e "${RED}Failed to start containers!${NC}"
    exit 1
fi

# Status check
echo ""
echo -e "${BLUE}Checking container status...${NC}"
sleep 3
$DOCKER_COMPOSE_CMD ps

# Show logs
echo ""
echo -e "${BLUE}Recent logs:${NC}"
$DOCKER_COMPOSE_CMD logs --tail=20

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Deployment complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${BLUE}Access the web interface:${NC}"
echo -e "  ${GREEN}http://localhost:$PORT${NC}"
echo ""
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    SERVER_IP=$(hostname -I | awk '{print $1}' 2>/dev/null || echo "server-ip")
    echo -e "${BLUE}Server IP:${NC}"
    echo -e "  ${GREEN}http://${SERVER_IP}:$PORT${NC}"
    echo ""
fi
echo -e "${BLUE}Useful commands:${NC}"
echo -e "  View logs:          ${YELLOW}$DOCKER_COMPOSE_CMD logs -f${NC}"
echo -e "  Container status:   ${YELLOW}$DOCKER_COMPOSE_CMD ps${NC}"
echo -e "  Stop containers:    ${YELLOW}$DOCKER_COMPOSE_CMD down${NC}"
echo -e "  Restart containers: ${YELLOW}$DOCKER_COMPOSE_CMD restart${NC}"
echo ""
