# Quick Fix for EC2 Deployment

## Problem
- `ECR_REGISTRY` not set
- Services not running
- Can't connect to port 3000

## Solution: Two Options

---

## Option 1: Use ECR Images (Recommended - Saves Disk Space)

### Step 1: Get your AWS Account ID
```bash
aws sts get-caller-identity --query Account --output text
```

### Step 2: Set ECR_REGISTRY
```bash
# Replace YOUR_ACCOUNT_ID with your actual account ID
export ECR_REGISTRY=YOUR_ACCOUNT_ID.dkr.ecr.ap-south-1.amazonaws.com

# Verify it's set
echo $ECR_REGISTRY
```

### Step 3: Login to ECR
```bash
aws ecr get-login-password --region ap-south-1 | docker login --username AWS --password-stdin $ECR_REGISTRY
```

### Step 4: Pull and start services
```bash
cd ~/sherlock-service
docker-compose -f docker/docker-compose.ecr.yml pull
docker-compose -f docker/docker-compose.ecr.yml up -d
```

### Step 5: Verify
```bash
docker-compose -f docker/docker-compose.ecr.yml ps
curl http://localhost:3000/health
```

---

## Option 2: Build Locally (If ECR Not Available)

### Step 1: Use the build compose file
```bash
cd ~/sherlock-service
docker-compose -f docker/docker-compose.t3micro.yml up -d --build
```

### Step 2: Verify
```bash
docker-compose -f docker/docker-compose.t3micro.yml ps
curl http://localhost:3000/health
```

**Note**: This will use more disk space (~3GB) but works without ECR setup.

---

## Make ECR_REGISTRY Permanent

Add to your `.bashrc` or `.profile`:

```bash
# Add to ~/.bashrc
echo 'export ECR_REGISTRY=YOUR_ACCOUNT_ID.dkr.ecr.ap-south-1.amazonaws.com' >> ~/.bashrc
source ~/.bashrc
```

Or add to `.env` file:

```bash
# Add to ~/sherlock-service/.env
echo 'ECR_REGISTRY=YOUR_ACCOUNT_ID.dkr.ecr.ap-south-1.amazonaws.com' >> ~/sherlock-service/.env
```

---

## Troubleshooting

### Issue: "repository name must be lowercase"
**Fix**: Make sure ECR_REGISTRY doesn't have uppercase letters

### Issue: "unauthorized: authentication required"
**Fix**: Run ECR login command again

### Issue: "No such image"
**Fix**: Images need to be built and pushed to ECR first (via GitHub Actions)

### Issue: Still can't connect to port 3000
**Fix**: 
```bash
# Check if services are running
docker-compose -f docker/docker-compose.ecr.yml ps

# Check logs
docker-compose -f docker/docker-compose.ecr.yml logs

# Restart services
docker-compose -f docker/docker-compose.ecr.yml restart
```

---

## Quick Commands

```bash
# Set ECR registry (replace YOUR_ACCOUNT_ID)
export ECR_REGISTRY=YOUR_ACCOUNT_ID.dkr.ecr.ap-south-1.amazonaws.com

# Login to ECR
aws ecr get-login-password --region ap-south-1 | docker login --username AWS --password-stdin $ECR_REGISTRY

# Start services
cd ~/sherlock-service
docker-compose -f docker/docker-compose.ecr.yml up -d

# Check status
docker-compose -f docker/docker-compose.ecr.yml ps
curl http://localhost:3000/health
```

