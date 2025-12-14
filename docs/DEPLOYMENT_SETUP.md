# Deployment Setup - One-Time vs Every Time

## One-Time Setup (Do Once) ✅

### Step 1: Get AWS Account ID
```bash
aws sts get-caller-identity --query Account --output text
# Save this number: 637423495478
```

### Step 2: Make ECR_REGISTRY Permanent
```bash
# Add to .env file (persists across deployments)
cd ~/sherlock-service
echo 'ECR_REGISTRY=637423495478.dkr.ecr.ap-south-1.amazonaws.com' >> .env

# Or add to .bashrc (persists across sessions)
echo 'export ECR_REGISTRY=637423495478.dkr.ecr.ap-south-1.amazonaws.com' >> ~/.bashrc
source ~/.bashrc
```

**Done!** This is permanent. You won't need to set it again.

---

## Automated Deployment (GitHub Actions) 🚀

Once you've set up GitHub secrets, deployments are **fully automated**:

1. **Push to GitHub** → `git push origin main`
2. **GitHub Actions automatically**:
   - Builds Docker images
   - Pushes to ECR
   - Logs into ECR
   - Pulls latest code on EC2
   - Pulls images from ECR
   - Restarts services
   - Runs health checks

**You don't need to do anything!** Just push code.

---

## Manual Deployment (If Needed)

If you want to deploy manually (without GitHub Actions):

### Quick Manual Deploy:
```bash
cd ~/sherlock-service
git pull origin main
docker-compose -f docker/docker-compose.ecr.yml pull
docker-compose -f docker/docker-compose.ecr.yml up -d
```

**That's it!** Only 3 commands because ECR_REGISTRY is already set.

---

## What Happens When

| Action | Frequency | Who Does It |
|--------|-----------|-------------|
| **Set ECR_REGISTRY** | Once | You (one-time) |
| **ECR Login** | Auto (GitHub Actions) | GitHub Actions |
| **Pull Code** | Every deploy | GitHub Actions (auto) |
| **Pull Images** | Every deploy | GitHub Actions (auto) |
| **Restart Services** | Every deploy | GitHub Actions (auto) |

---

## Summary

### ✅ One-Time Setup:
1. Get AWS Account ID
2. Set ECR_REGISTRY in `.env` or `.bashrc`
3. Configure GitHub Secrets (EC2_HOST, EC2_SSH_KEY, AWS keys)

### 🚀 Every Deployment (Automated):
- Just push: `git push origin main`
- GitHub Actions does everything else!

### 🔧 Manual Deploy (If Needed):
```bash
git pull && docker-compose -f docker/docker-compose.ecr.yml pull && docker-compose -f docker/docker-compose.ecr.yml up -d
```

---

## Make It Permanent Right Now

Run this once on EC2:

```bash
cd ~/sherlock-service

# Add ECR_REGISTRY to .env (docker-compose reads this)
echo 'ECR_REGISTRY=637423495478.dkr.ecr.ap-south-1.amazonaws.com' >> .env

# Verify it's set
cat .env | grep ECR_REGISTRY
```

Now `docker-compose.ecr.yml` will automatically use it! No need to export every time.

---

**Bottom Line**: Set it once, then deployments are automatic! 🎉

