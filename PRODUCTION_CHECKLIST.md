# Production Verification Checklist

## ✅ Pre-Deployment Checks

- [x] Services deployed to EC2
- [x] Docker Compose running
- [x] ECR images pulling correctly
- [x] GitHub Actions deployment working

---

## 🧪 Testing Checklist

### 1. Basic Health Checks

- [ ] **Health endpoint**
  ```bash
  curl http://localhost:3000/health
  # Expected: {"status":"ok"}
  ```

- [ ] **Readiness check**
  ```bash
  curl http://localhost:3000/health/ready
  # Expected: {"status":"ready"}
  ```

- [ ] **Services status**
  ```bash
  docker-compose -f docker/docker-compose.ecr.yml ps
  # Expected: All services "Up"
  ```

---

### 2. Database & Redis

- [ ] **PostgreSQL connection**
  ```bash
  docker-compose -f docker/docker-compose.ecr.yml exec postgres psql -U sherlock -d sherlock -c "SELECT 1;"
  # Expected: Returns 1
  ```

- [ ] **Redis connection**
  ```bash
  docker-compose -f docker/docker-compose.ecr.yml exec redis redis-cli ping
  # Expected: PONG
  ```

- [ ] **Migrations applied**
  ```bash
  docker-compose -f docker/docker-compose.ecr.yml exec postgres psql -U sherlock -d sherlock -c "\d reviews"
  # Expected: Shows reviews table with indexes
  ```

---

### 3. API Endpoints

- [ ] **Stats endpoint**
  ```bash
  curl http://localhost:3000/api/v1/stats \
    -H "X-Org-ID: test-org"
  # Expected: Returns stats JSON
  ```

- [ ] **Metrics endpoint** (if implemented)
  ```bash
  curl http://localhost:3000/api/metrics
  # Expected: Returns metrics JSON
  ```

---

### 4. Webhook & Review Flow

- [ ] **Webhook receives events**
  - Create a test PR
  - Check webhook logs
  - Expected: Webhook received and processed

- [ ] **Review job created**
  ```bash
  # Check database
  docker-compose -f docker/docker-compose.ecr.yml exec postgres psql -U sherlock -d sherlock -c "SELECT * FROM reviews ORDER BY created_at DESC LIMIT 1;"
  # Expected: New review record
  ```

- [ ] **Review completes**
  - Check review status
  - Expected: Status = "completed"

- [ ] **Cache working**
  ```bash
  # Check Redis for cache keys
  docker-compose -f docker/docker-compose.ecr.yml exec redis redis-cli --scan --pattern "review:cache:*"
  # Expected: Cache keys present
  ```

---

### 5. Performance Checks

- [ ] **Review time**
  - Trigger a review
  - Measure time to completion
  - Expected: < 1 minute (with incremental)

- [ ] **Cache hit rate**
  - Trigger same review twice
  - Check logs for "Cache hit"
  - Expected: Second review uses cache

- [ ] **Disk space**
  ```bash
  df -h
  docker system df
  # Expected: < 50% disk usage
  ```

- [ ] **Memory usage**
  ```bash
  docker stats
  # Expected: All containers < memory limits
  ```

---

### 6. Error Handling

- [ ] **Invalid webhook**
  - Send malformed webhook
  - Expected: Graceful error, no crash

- [ ] **Database connection loss**
  - Simulate DB down
  - Expected: Retry logic, graceful degradation

- [ ] **Redis connection loss**
  - Simulate Redis down
  - Expected: Falls back, no crash

---

## 📊 Metrics to Track

### Daily Monitoring

- [ ] Cache hit rate: Target > 50%
- [ ] Average review time: Target < 1 minute
- [ ] Success rate: Target > 90%
- [ ] Error rate: Target < 5%
- [ ] Disk usage: Target < 50%

### Weekly Review

- [ ] Total reviews processed
- [ ] Cost per review
- [ ] User feedback (if available)
- [ ] Performance trends

---

## 🚨 Alerts to Set Up

- [ ] Disk usage > 80%
- [ ] Service down
- [ ] Error rate > 10%
- [ ] Review time > 2 minutes
- [ ] Database connection failures

---

## ✅ Sign-Off

- [ ] All health checks passing
- [ ] Webhook flow working
- [ ] Reviews completing successfully
- [ ] Performance metrics acceptable
- [ ] Error handling verified
- [ ] Monitoring in place

**Status**: ⬜ Not Started | 🟡 In Progress | ✅ Complete

**Date**: ___________

**Verified by**: ___________
