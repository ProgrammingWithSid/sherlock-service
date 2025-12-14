# How to Get ECR_REGISTRY Value

## Quick Answer

Your ECR_REGISTRY is: `YOUR_ACCOUNT_ID.dkr.ecr.ap-south-1.amazonaws.com`

You just need your **AWS Account ID** (12-digit number).

---

## Method 1: Get from AWS CLI (Easiest)

### On EC2 or your local machine:

```bash
# Get your AWS Account ID
aws sts get-caller-identity --query Account --output text

# Output example: 637423495478

# Your ECR_REGISTRY is:
# 637423495478.dkr.ecr.ap-south-1.amazonaws.com
```

**That's it!** Replace `637423495478` with your account ID.

---

## Method 2: Get from AWS Console

1. Go to: https://console.aws.amazon.com/
2. Click your username (top right)
3. Your **Account ID** is shown there (12-digit number)
4. Your ECR_REGISTRY is: `ACCOUNT_ID.dkr.ecr.ap-south-1.amazonaws.com`

---

## Method 3: Get from ECR Console

1. Go to: https://console.aws.amazon.com/ecr/
2. Click on any repository
3. Click **"View push commands"**
4. The registry URL is shown at the top:
   ```
   637423495478.dkr.ecr.ap-south-1.amazonaws.com
   ```

---

## Method 4: From GitHub Actions (If Already Working)

If your GitHub Actions workflow is already working, check the logs:

1. Go to: GitHub → Actions → Latest workflow run
2. Look for: `🔑 ECR Registry: ...`
3. Copy that value

---

## Format

ECR_REGISTRY format is always:
```
ACCOUNT_ID.dkr.ecr.REGION.amazonaws.com
```

For your setup:
- **Account ID**: Your 12-digit AWS account number
- **Region**: `ap-south-1` (Mumbai)
- **Full format**: `ACCOUNT_ID.dkr.ecr.ap-south-1.amazonaws.com`

---

## Example

If your Account ID is `637423495478`:
```
ECR_REGISTRY=637423495478.dkr.ecr.ap-south-1.amazonaws.com
```

---

## Verify It's Correct

```bash
# Test ECR login
aws ecr get-login-password --region ap-south-1 | \
  docker login --username AWS --password-stdin \
  637423495478.dkr.ecr.ap-south-1.amazonaws.com

# If this works, your ECR_REGISTRY is correct!
```

---

## Quick Command to Get It

```bash
# One-liner to get your ECR_REGISTRY
echo "$(aws sts get-caller-identity --query Account --output text).dkr.ecr.ap-south-1.amazonaws.com"
```

**Output**: Your complete ECR_REGISTRY value ready to use!

---

## Common Values

Based on your Makefile, your Account ID appears to be: `637423495478`

So your ECR_REGISTRY is:
```
637423495478.dkr.ecr.ap-south-1.amazonaws.com
```

---

## Where to Use It

### Option 1: In .env file (Recommended)
```bash
echo 'ECR_REGISTRY=637423495478.dkr.ecr.ap-south-1.amazonaws.com' >> ~/sherlock-service/.env
```

### Option 2: Export in shell
```bash
export ECR_REGISTRY=637423495478.dkr.ecr.ap-south-1.amazonaws.com
```

### Option 3: Already in docker-compose.ecr.yml
The file already has a default value, so you don't need to set it if you're using that account ID!

---

**That's it!** Your ECR_REGISTRY is just your Account ID + `.dkr.ecr.ap-south-1.amazonaws.com`

