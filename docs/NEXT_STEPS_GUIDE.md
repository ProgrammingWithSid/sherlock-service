# Next Steps Guide - Where to Do What

## 🎯 Quick Overview

1. **On EC2** → Clean up disk space
2. **On EC2** → Setup ECR authentication
3. **On GitHub** → Add secrets for automated deployment
4. **Test** → Deploy and verify

---

## Step 1: Clean Up Disk Space (On EC2)

### Where: SSH into your EC2 instance

```bash
# Connect to EC2
ssh ubuntu@your-ec2-ip

# Navigate to project
cd sherlock-service

# Run cleanup script
./scripts/docker-cleanup.sh
```

**Expected result**: Frees 1-2GB disk space

---

## Step 2: Setup ECR Authentication (On EC2)

### Where: Same EC2 instance (still SSH'd in)

```bash
# Still on EC2, in sherlock-service directory

# Get your AWS Account ID (if you don't know it)
aws sts get-caller-identity --query Account --output text

# Setup ECR auth (replace YOUR_ACCOUNT_ID)
./scripts/setup-ecr-auth.sh ap-south-1 YOUR_ACCOUNT_ID
```

**Expected result**: Docker can now pull from ECR

---

## Step 3: Configure GitHub Secrets (On GitHub)

### Where: GitHub Repository Settings

1. **Go to your GitHub repo**: `https://github.com/YOUR_USERNAME/sherlock-service`

2. **Click**: Settings → Secrets and variables → Actions

3. **Add these secrets**:

   | Secret Name | Value | Where to Find |
   |------------|-------|---------------|
   | `EC2_HOST` | Your EC2 IP (e.g., `54.123.45.67`) | EC2 Console → Instances → Public IPv4 |
   | `EC2_SSH_KEY` | Your SSH private key | `~/.ssh/id_rsa` or `~/.ssh/id_ed25519` |
   | `AWS_ACCESS_KEY_ID` | AWS access key | AWS Console → IAM → Users → Security credentials |
   | `AWS_SECRET_ACCESS_KEY` | AWS secret key | Same as above (create if needed) |

### How to get SSH key:

```bash
# On your local machine
cat ~/.ssh/id_rsa
# Copy entire output (including -----BEGIN and -----END lines)
```

### How to create AWS credentials:

1. AWS Console → IAM → Users → Your user
2. Security credentials → Create access key
3. Copy Access Key ID and Secret Access Key

---

## Step 4: Test Deployment

### Option A: Manual Test (On EC2)

```bash
# Still SSH'd into EC2

# Set ECR registry (replace YOUR_ACCOUNT_ID)
export ECR_REGISTRY=YOUR_ACCOUNT_ID.dkr.ecr.ap-south-1.amazonaws.com

# Stop old containers
docker-compose -f docker/docker-compose.t3micro.yml down

# Start with ECR images
docker-compose -f docker/docker-compose.ecr.yml up -d

# Check status
docker-compose -f docker/docker-compose.ecr.yml ps

# Check logs
docker-compose -f docker/docker-compose.ecr.yml logs -f
```

### Option B: Automated Test (GitHub Actions)

1. **Make a small change** (or just push current code):
   ```bash
   # On your local machine
   git add .
   git commit -m "test: ECR deployment"
   git push origin main
   ```

2. **Watch GitHub Actions**:
   - Go to: `https://github.com/YOUR_USERNAME/sherlock-service/actions`
   - Click on the running workflow
   - Watch it build images → push to ECR → deploy to EC2

---

## Step 5: Verify Everything Works

### On EC2 (SSH):

```bash
# Check services are running
docker-compose -f docker/docker-compose.ecr.yml ps

# Check disk space (should be much better now)
df -h

# Check Docker disk usage (should be minimal)
docker system df

# Test health endpoint
curl http://localhost:3000/health
```

### Expected Results:

- ✅ Services running
- ✅ Disk usage < 50%
- ✅ Docker using < 1GB
- ✅ Health check returns 200 OK

---

## Troubleshooting

### Issue: "ECR authentication failed"

**Fix**:
```bash
# On EC2
aws ecr get-login-password --region ap-south-1 | docker login --username AWS --password-stdin YOUR_ACCOUNT_ID.dkr.ecr.ap-south-1.amazonaws.com
```

### Issue: "Cannot connect to EC2"

**Fix**:
- Check EC2 security group allows SSH (port 22) from your IP
- Verify EC2 instance is running
- Check SSH key is correct

### Issue: "GitHub Actions deployment failed"

**Fix**:
- Verify all secrets are set correctly
- Check GitHub Actions logs for specific error
- Ensure AWS credentials have ECR permissions

### Issue: "Still running out of disk space"

**Fix**:
```bash
# On EC2 - More aggressive cleanup
docker system prune -a -f --volumes
docker builder prune -a -f

# Or increase disk (last resort)
# AWS Console → EC2 → Volumes → Modify Volume → Increase size
```

---

## Quick Reference: File Locations

| Task | File Location | Where to Run |
|------|--------------|--------------|
| Cleanup script | `scripts/docker-cleanup.sh` | EC2 (SSH) |
| ECR auth script | `scripts/setup-ecr-auth.sh` | EC2 (SSH) |
| ECR compose file | `docker/docker-compose.ecr.yml` | EC2 (SSH) |
| GitHub workflow | `.github/workflows/deploy-simple.yml` | GitHub (auto) |
| GitHub secrets | Settings → Secrets | GitHub (web) |

---

## Checklist

- [ ] Step 1: Cleaned up disk space on EC2
- [ ] Step 2: Setup ECR authentication on EC2
- [ ] Step 3: Added GitHub secrets (EC2_HOST, EC2_SSH_KEY, AWS keys)
- [ ] Step 4: Tested manual deployment
- [ ] Step 5: Verified services are running
- [ ] Step 6: Tested automated deployment (push to GitHub)

---

## Summary: Where to Do What

| Step | Location | Command/Action |
|------|----------|---------------|
| **Cleanup** | EC2 (SSH) | `./scripts/docker-cleanup.sh` |
| **ECR Setup** | EC2 (SSH) | `./scripts/setup-ecr-auth.sh ap-south-1 ACCOUNT_ID` |
| **Add Secrets** | GitHub (Web) | Settings → Secrets → Actions → New secret |
| **Test Deploy** | EC2 (SSH) | `docker-compose -f docker/docker-compose.ecr.yml up -d` |
| **Auto Deploy** | GitHub (Push) | `git push origin main` |

---

**Need help?** Check the logs:
- EC2 logs: `docker-compose -f docker/docker-compose.ecr.yml logs -f`
- GitHub Actions: Repository → Actions tab

