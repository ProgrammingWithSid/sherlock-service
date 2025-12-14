# Troubleshooting 502 Bad Gateway

## Problem
Nginx returns 502 Bad Gateway, meaning it can't connect to the backend service.

## Quick Checks

### 1. Check if services are running
```bash
cd sherlock-service
docker-compose -f docker/docker-compose.ecr.yml ps
```

All services should show "Up" status.

### 2. Check backend health directly
```bash
curl http://localhost:3000/health
```

Should return: `{"status":"ok"}`

### 3. Check service logs
```bash
docker-compose -f docker/docker-compose.ecr.yml logs api
docker-compose -f docker/docker-compose.ecr.yml logs worker
```

Look for errors or startup issues.

## Common Issues & Fixes

### Issue 1: Services not started
**Symptom:** `docker-compose ps` shows services as "Exit" or not listed

**Fix:**
```bash
cd sherlock-service
export ECR_REGISTRY=637423495478.dkr.ecr.ap-south-1.amazonaws.com
docker-compose -f docker/docker-compose.ecr.yml up -d
```

### Issue 2: Backend service crashed
**Symptom:** Backend shows "Exit" status in `docker-compose ps`

**Fix:**
```bash
# Check logs for errors
docker-compose -f docker/docker-compose.ecr.yml logs api

# Common causes:
# - Database connection failed
# - Missing environment variables
# - Port already in use

# Restart service
docker-compose -f docker/docker-compose.ecr.yml restart api
```

### Issue 3: Nginx proxy misconfiguration
**Symptom:** Nginx is running but backend isn't accessible

**Check nginx config:**
```bash
sudo cat /etc/nginx/sites-available/default
# or
sudo cat /etc/nginx/nginx.conf
```

**Nginx should proxy to:**
```nginx
proxy_pass http://localhost:3000;
```

**Fix nginx config if needed:**
```bash
sudo nano /etc/nginx/sites-available/default
# Update proxy_pass to: http://localhost:3000;
sudo nginx -t  # Test config
sudo systemctl reload nginx
```

### Issue 4: Port conflict
**Symptom:** Port 3000 already in use

**Check:**
```bash
sudo netstat -tlnp | grep 3000
# or
sudo ss -tlnp | grep 3000
```

**Fix:**
```bash
# Stop conflicting service or change port in docker-compose.yml
docker-compose -f docker/docker-compose.ecr.yml down
docker-compose -f docker/docker-compose.ecr.yml up -d
```

### Issue 5: Database/Redis not ready
**Symptom:** Backend can't connect to database

**Fix:**
```bash
# Check database is running
docker-compose -f docker/docker-compose.ecr.yml ps postgres redis

# Check database logs
docker-compose -f docker/docker-compose.ecr.yml logs postgres

# Restart database if needed
docker-compose -f docker/docker-compose.ecr.yml restart postgres redis
```

## Diagnostic Script

Run the diagnostic script:
```bash
cd sherlock-service
./scripts/check-services.sh
```

This will check:
- Docker container status
- Service health endpoints
- Port availability
- Recent logs
- Nginx status

## Manual Service Restart

If all else fails, restart all services:
```bash
cd sherlock-service
export ECR_REGISTRY=637423495478.dkr.ecr.ap-south-1.amazonaws.com

# Stop all services
docker-compose -f docker/docker-compose.ecr.yml down

# Start all services
docker-compose -f docker/docker-compose.ecr.yml up -d

# Wait for services to start
sleep 15

# Check status
docker-compose -f docker/docker-compose.ecr.yml ps

# Test backend
curl http://localhost:3000/health
```

## Verify Nginx Configuration

If using nginx as reverse proxy:

1. **Check nginx is running:**
   ```bash
   sudo systemctl status nginx
   ```

2. **Check nginx config:**
   ```bash
   sudo nginx -t
   ```

3. **View nginx error logs:**
   ```bash
   sudo tail -f /var/log/nginx/error.log
   ```

4. **Example nginx config:**
   ```nginx
   server {
       listen 80;
       server_name your-domain.com;

       location / {
           proxy_pass http://localhost:3000;
           proxy_http_version 1.1;
           proxy_set_header Upgrade $http_upgrade;
           proxy_set_header Connection 'upgrade';
           proxy_set_header Host $host;
           proxy_cache_bypass $http_upgrade;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
       }
   }
   ```

## Still Not Working?

1. **Check EC2 security groups** - Ensure ports 80, 3000 are open
2. **Check firewall** - `sudo ufw status`
3. **Check service logs** - `docker-compose logs -f`
4. **Verify environment variables** - Check `backend/.env` file exists and has correct values

