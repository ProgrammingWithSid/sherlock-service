#!/bin/bash
#
# Update EC2 Instance for New Monorepo Structure
# Pulls latest code and rebuilds Docker containers
#

set -e

EC2_IP="13.233.117.33"
EC2_USER="ubuntu"
PEM_KEY="${HOME}/Desktop/sherlock-service.pem"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🔄 Updating EC2 instance for new monorepo structure...${NC}"
echo ""

# Check if PEM key exists
if [ ! -f "$PEM_KEY" ]; then
    echo -e "${YELLOW}⚠️  PEM key not found at: ${PEM_KEY}${NC}"
    read -p "Enter path to PEM key: " PEM_KEY
fi

if [ ! -f "$PEM_KEY" ]; then
    echo -e "${RED}❌ PEM key not found: ${PEM_KEY}${NC}"
    exit 1
fi

chmod 400 "$PEM_KEY"

echo -e "${YELLOW}📡 Connecting to EC2 instance...${NC}"
echo ""

# Create update script to run on EC2
cat > /tmp/update-ec2.sh << 'EC2SCRIPT'
#!/bin/bash
set -e

echo "🔄 Updating sherlock-service on EC2..."
echo ""

# Navigate to project directory
cd ~/sherlock-service || {
    echo "❌ sherlock-service directory not found!"
    exit 1
}

# Backup current state
echo "📦 Backing up current state..."
if [ -d "docker" ]; then
    cd docker
    docker-compose ps > /tmp/docker-status.txt 2>/dev/null || true
    cd ..
fi

# Pull latest code
echo "📥 Pulling latest code from git..."

# Check for local changes
if ! git diff --quiet || ! git diff --cached --quiet; then
    echo "⚠️  Local changes detected. Stashing them..."
    git stash push -m "Stash before monorepo update $(date +%Y%m%d_%H%M%S)"
fi

# Handle untracked files that might conflict
if [ -f "frontend/.env.production" ] && ! git ls-files --error-unmatch frontend/.env.production >/dev/null 2>&1; then
    echo "⚠️  Backing up untracked frontend/.env.production..."
    mv frontend/.env.production frontend/.env.production.backup 2>/dev/null || true
fi

if [ -f "package.json" ] && ! git ls-files --error-unmatch package.json >/dev/null 2>&1; then
    echo "⚠️  Backing up untracked package.json..."
    mv package.json package.json.backup 2>/dev/null || true
fi

# Pull latest code
git pull origin main || {
    echo "⚠️  Git pull failed. Trying to reset to remote..."
    git fetch origin main
    git reset --hard origin/main || {
        echo "❌ Failed to update code. Please resolve conflicts manually."
        exit 1
    }
}

# Check if backend folder exists
if [ ! -d "backend" ]; then
    echo "❌ backend/ folder not found after git pull!"
    echo "   Current structure:"
    ls -la
    exit 1
fi

echo "✅ Code updated successfully"
echo ""

# Rebuild Docker containers
echo "🔨 Rebuilding Docker containers..."
cd docker

# Stop containers
echo "   Stopping containers..."
docker-compose down || true

# Rebuild with new structure
echo "   Building new images..."
docker-compose build --no-cache

# Start containers
echo "   Starting containers..."
docker-compose up -d

# Wait for services to start
sleep 5

# Check status
echo ""
echo "📊 Container status:"
docker-compose ps

echo ""
echo "✅ Update complete!"
echo ""
echo "🧪 Test your services:"
echo "   curl http://localhost:3000/health"
EC2SCRIPT

# Copy update script to EC2
echo -e "${BLUE}📤 Copying update script to EC2...${NC}"
scp -i "$PEM_KEY" /tmp/update-ec2.sh ${EC2_USER}@${EC2_IP}:/tmp/update-ec2.sh

# Run update script on EC2
echo -e "${BLUE}🚀 Running update script on EC2...${NC}"
ssh -i "$PEM_KEY" ${EC2_USER}@${EC2_IP} "chmod +x /tmp/update-ec2.sh && /tmp/update-ec2.sh"

# Cleanup
rm -f /tmp/update-ec2.sh

echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}✅ EC2 update complete!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${BLUE}📝 What was updated:${NC}"
echo -e "   ✅ Pulled latest code with backend/ folder structure"
echo -e "   ✅ Rebuilt Docker containers with new paths"
echo -e "   ✅ Restarted services"
echo ""
echo -e "${BLUE}🔗 Test your services:${NC}"
echo -e "   Backend: https://api.algovesh.com/health"
echo -e "   Frontend: https://app.algovesh.com"
echo ""
