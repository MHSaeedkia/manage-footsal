#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}==================================${NC}"
echo -e "${GREEN}Futsal Bot Setup Script${NC}"
echo -e "${GREEN}==================================${NC}"
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    echo "Please install Docker first: https://docs.docker.com/get-docker/"
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}Error: Docker Compose is not installed${NC}"
    echo "Please install Docker Compose first: https://docs.docker.com/compose/install/"
    exit 1
fi

echo -e "${GREEN}✓${NC} Docker is installed"
echo -e "${GREEN}✓${NC} Docker Compose is installed"
echo ""

# Check if .env file exists
if [ -f .env ]; then
    echo -e "${YELLOW}⚠${NC}  .env file already exists"
    read -p "Do you want to overwrite it? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Keeping existing .env file"
    else
        cp .env.example .env
        echo -e "${GREEN}✓${NC} .env file created"
    fi
else
    cp .env.example .env
    echo -e "${GREEN}✓${NC} .env file created"
fi

echo ""
echo -e "${YELLOW}Please configure the following in .env file:${NC}"
echo "1. BOT_TOKEN - Get it from @BotFather on Telegram"
echo "2. DEFAULT_ADMIN_ID - Your Telegram user ID (get from @userinfobot)"
echo "3. DB_PASSWORD - A secure password for PostgreSQL"
echo ""

read -p "Have you configured the .env file? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Please edit .env file and run this script again${NC}"
    echo "You can edit it with: nano .env"
    exit 0
fi

# Verify required env vars
source .env

if [ -z "$BOT_TOKEN" ] || [ "$BOT_TOKEN" = "your_telegram_bot_token_here" ]; then
    echo -e "${RED}Error: BOT_TOKEN is not configured in .env${NC}"
    exit 1
fi

if [ -z "$DEFAULT_ADMIN_ID" ] || [ "$DEFAULT_ADMIN_ID" = "your_telegram_user_id_here" ]; then
    echo -e "${RED}Error: DEFAULT_ADMIN_ID is not configured in .env${NC}"
    exit 1
fi

if [ -z "$DB_PASSWORD" ] || [ "$DB_PASSWORD" = "your_secure_password_here" ]; then
    echo -e "${RED}Error: DB_PASSWORD is not configured in .env${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Configuration verified"
echo ""

# Build and start services
echo "Building Docker images..."
docker-compose build

if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Failed to build Docker images${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Docker images built successfully"
echo ""

echo "Starting services..."
docker-compose up -d

if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Failed to start services${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Services started successfully"
echo ""

# Wait for services to be ready
echo "Waiting for services to be ready..."
sleep 5

# Check service status
echo ""
echo "Service status:"
docker-compose ps

echo ""
echo -e "${GREEN}==================================${NC}"
echo -e "${GREEN}Setup completed successfully!${NC}"
echo -e "${GREEN}==================================${NC}"
echo ""
echo "Next steps:"
echo "1. Add your bot to a Telegram group"
echo "2. Send /start to your bot in private chat"
echo "3. Register yourself in the bot"
echo ""
echo "Useful commands:"
echo "  docker-compose logs -f        # View logs"
echo "  docker-compose logs -f bot    # View bot logs only"
echo "  docker-compose restart        # Restart services"
echo "  docker-compose down           # Stop services"
echo ""
echo "For more information, see README.md and QUICKSTART.md"
