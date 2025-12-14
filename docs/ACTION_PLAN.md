# Action Plan - What to Do Next

## 🎯 Current Status

✅ Simplified deployment (Docker Compose instead of ECS)  
✅ Added ECR support (saves disk space)  
✅ Cleaned up unnecessary files  
✅ Created deployment scripts and guides  

## 🚀 Next Steps (In Order)

---

## Step 1: Deploy to EC2 (If Not Already Done)

### On EC2 Instance:

```bash
# SSH into EC2
ssh ubuntu@your-ec2-ip

# Clone repository (if not already)
git clone <your-repo-url> sherlock-service
cd sherlock-service

# Run initial setup
chmod +x scripts/setup-ec2.sh
./scripts/setup-ec2.sh

# Edit .env file with your values
nano .env
# Add: GitHub credentials, AI API keys, etc.

# Clean up disk space
./scripts/docker-cleanup.sh

# Setup ECR authentication
./scripts/setup-ecr-auth.sh ap-south-1 YOUR_ACCOUNT_ID

# Deploy using ECR images
export ECR_REGISTRY=YOUR_ACCOUNT_ID.dkr.ecr.ap-south-1.amazonaws.com
docker-compose -f docker/docker-compose.ecr.yml up -d

# Verify
docker-compose -f docker/docker-compose.ecr.yml ps
curl http://localhost:3000/health
```

**Expected time**: 10-15 minutes  
**Result**: Services running on EC2

---

## Step 2: Configure GitHub Secrets (For Automated Deployment)

### On GitHub Website:

1. Go to: `https://github.com/YOUR_USERNAME/sherlock-service`
2. Click: **Settings** → **Secrets and variables** → **Actions**
3. Click: **New repository secret**

Add these 4 secrets:

| Secret Name | How to Get |
|------------|------------|
| `EC2_HOST` | EC2 Console → Instances → Public IPv4 address |
| `EC2_SSH_KEY` | On your local machine: `cat ~/.ssh/id_rsa` (copy entire output) |
| `AWS_ACCESS_KEY_ID` | AWS Console → IAM → Users → Your user → Security credentials → Create access key |
| `AWS_SECRET_ACCESS_KEY` | Same as above (shown only once, save it!) |

**Expected time**: 5 minutes  
**Result**: Automated deployments enabled

---

## Step 3: Test Automated Deployment

### On Your Local Machine:

```bash
# Make a small change (or just push current code)
cd sherlock-service
git add .
git commit -m "test: automated deployment"
git push origin main
```

### On GitHub:

1. Go to: **Actions** tab
2. Watch the workflow run:
   - ✅ Builds Docker images
   - ✅ Pushes to ECR
   - ✅ Deploys to EC2
   - ✅ Health check

**Expected time**: 5-10 minutes (first time)  
**Result**: Automated deployment working

---

## Step 4: Verify Everything Works

### On EC2 (SSH):

```bash
# Check services
docker-compose -f docker/docker-compose.ecr.yml ps

# Check logs
docker-compose -f docker/docker-compose.ecr.yml logs -f

# Check disk space (should be good now)
df -h
docker system df

# Test API
curl http://localhost:3000/health
curl http://localhost:3000/api/v1/stats
```

### Test Features:

1. **Webhook**: Create a test PR, verify review triggers
2. **Caching**: Check logs for "Cache hit" messages
3. **Incremental Reviews**: Verify faster review times
4. **Metrics**: Check `/api/metrics` endpoint

**Expected time**: 10 minutes  
**Result**: Everything verified and working

---

## Step 5: Set Up Domain & SSL (Optional)

### If you have a domain:

1. **Point DNS** to EC2 IP:
   ```
   A record: your-domain.com → EC2_IP
   ```

2. **Install SSL** (Let's Encrypt):
   ```bash
   # On EC2
   sudo apt-get install certbot
   sudo certbot certonly --standalone -d your-domain.com
   ```

3. **Update nginx/Caddy** to use SSL

**Expected time**: 15 minutes  
**Result**: HTTPS enabled

---

## Step 6: Monitor & Optimize

### Set Up Monitoring:

1. **Check logs regularly**:
   ```bash
   docker-compose -f docker/docker-compose.ecr.yml logs -f
   ```

2. **Monitor disk space**:
   ```bash
   df -h
   docker system df
   ```

3. **Check metrics**:
   ```bash
   curl http://localhost:3000/api/metrics
   ```

### Optimize:

- **Cache hit rate**: Should be > 50%
- **Review time**: Should be < 1 minute (with incremental)
- **Disk usage**: Should be < 50%
- **Cost**: Should be ~$10-20/month

---

## Quick Checklist

- [ ] **Step 1**: Deployed to EC2
- [ ] **Step 2**: Added GitHub secrets
- [ ] **Step 3**: Tested automated deployment
- [ ] **Step 4**: Verified all features work
- [ ] **Step 5**: Set up domain/SSL (optional)
- [ ] **Step 6**: Monitoring in place

---

## If Something Goes Wrong

### Services won't start:
```bash
# Check logs
docker-compose -f docker/docker-compose.ecr.yml logs

# Restart
docker-compose -f docker/docker-compose.ecr.yml restart
```

### Out of disk space:
```bash
# Clean up
./scripts/docker-cleanup.sh

# Use ECR images (if not already)
export ECR_REGISTRY=YOUR_ACCOUNT_ID.dkr.ecr.ap-south-1.amazonaws.com
docker-compose -f docker/docker-compose.ecr.yml up -d
```

### GitHub Actions failing:
- Check secrets are correct
- Verify EC2 security group allows SSH
- Check AWS credentials have ECR permissions

---

## Success Criteria

✅ Services running on EC2  
✅ Automated deployments working  
✅ Cache hit rate > 50%  
✅ Review time < 1 minute  
✅ Disk usage < 50%  
✅ Cost < $20/month  

---

## Timeline

- **Today**: Steps 1-3 (30 minutes)
- **This week**: Steps 4-5 (1 hour)
- **Ongoing**: Step 6 (monitoring)

**Total setup time**: ~2 hours  
**Result**: Fully automated, optimized deployment! 🚀

