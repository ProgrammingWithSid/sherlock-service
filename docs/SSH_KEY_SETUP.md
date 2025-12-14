# SSH Key Setup Guide

## Quick Answer

### If you already have an SSH key:

```bash
# On your local machine
cat ~/.ssh/id_rsa
# Copy the entire output (including -----BEGIN and -----END lines)
```

### If you don't have an SSH key:

```bash
# Generate a new SSH key
ssh-keygen -t rsa -b 4096 -C "github-actions-deploy"
# Save it as: ~/.ssh/id_rsa_github_actions
# Don't set a passphrase (or GitHub Actions won't be able to use it)
```

---

## Detailed Steps

### Option 1: Use Existing SSH Key (Recommended)

#### Step 1: Check if you have an SSH key

```bash
# Check for existing keys
ls -la ~/.ssh/

# Common key names:
# - id_rsa (RSA key)
# - id_ed25519 (Ed25519 key)
# - id_ecdsa (ECDSA key)
```

#### Step 2: Get your SSH private key

```bash
# If you have id_rsa
cat ~/.ssh/id_rsa

# Or if you have id_ed25519
cat ~/.ssh/id_ed25519

# Copy the ENTIRE output, including:
# -----BEGIN OPENSSH PRIVATE KEY-----
# ... (all the content) ...
# -----END OPENSSH PRIVATE KEY-----
```

#### Step 3: Add public key to EC2 (if not already added)

```bash
# Copy your public key to EC2
ssh-copy-id -i ~/.ssh/id_rsa.pub ubuntu@YOUR_EC2_IP

# Or manually add to EC2:
ssh ubuntu@YOUR_EC2_IP
mkdir -p ~/.ssh
nano ~/.ssh/authorized_keys
# Paste your public key (id_rsa.pub content)
chmod 600 ~/.ssh/authorized_keys
```

#### Step 4: Test SSH connection

```bash
# Test that you can SSH without password
ssh -i ~/.ssh/id_rsa ubuntu@YOUR_EC2_IP

# If this works, your key is correct!
```

---

### Option 2: Generate New SSH Key

#### Step 1: Generate a new key pair

```bash
# Generate RSA key (most compatible)
ssh-keygen -t rsa -b 4096 -C "github-actions-deploy" -f ~/.ssh/id_rsa_github_actions

# When prompted:
# - Passphrase: Press Enter (leave empty for GitHub Actions)
# - Confirm: Press Enter

# This creates:
# - ~/.ssh/id_rsa_github_actions (private key - keep secret!)
# - ~/.ssh/id_rsa_github_actions.pub (public key - safe to share)
```

#### Step 2: Add public key to EC2

```bash
# Copy public key to EC2
ssh-copy-id -i ~/.ssh/id_rsa_github_actions.pub ubuntu@YOUR_EC2_IP

# Or manually:
cat ~/.ssh/id_rsa_github_actions.pub
# Copy the output, then:
ssh ubuntu@YOUR_EC2_IP
mkdir -p ~/.ssh
echo "PASTE_PUBLIC_KEY_HERE" >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

#### Step 3: Get private key for GitHub

```bash
# Get the private key
cat ~/.ssh/id_rsa_github_actions

# Copy the ENTIRE output
```

---

## Add SSH Key to GitHub Secrets

### Step 1: Go to GitHub Repository

1. Go to: `https://github.com/YOUR_USERNAME/sherlock-service`
2. Click: **Settings** → **Secrets and variables** → **Actions**
3. Click: **New repository secret**

### Step 2: Add EC2_SSH_KEY Secret

- **Name**: `EC2_SSH_KEY`
- **Value**: Paste your **ENTIRE** private key (the output from `cat ~/.ssh/id_rsa`)

**Important**: 
- Include the `-----BEGIN` and `-----END` lines
- Include all the content between them
- Don't add extra spaces or newlines

### Step 3: Verify Format

Your secret should look like this:
```
-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAABlwAAAAdzc2gtcn
... (many lines of encoded content) ...
-----END OPENSSH PRIVATE KEY-----
```

---

## Troubleshooting

### Issue: "Permission denied (publickey)"

**Solution**:
1. Make sure public key is on EC2:
   ```bash
   ssh ubuntu@YOUR_EC2_IP
   cat ~/.ssh/authorized_keys
   # Should show your public key
   ```

2. Check permissions on EC2:
   ```bash
   ssh ubuntu@YOUR_EC2_IP
   chmod 700 ~/.ssh
   chmod 600 ~/.ssh/authorized_keys
   ```

### Issue: "Host key verification failed"

**Solution**: This is normal for first connection. The workflow handles this automatically.

### Issue: "No such file or directory" when using SSH key

**Solution**: Make sure you're using the **private key** (id_rsa), not the public key (id_rsa.pub)

### Issue: SSH works locally but not in GitHub Actions

**Solutions**:
1. Check EC2 security group allows SSH from anywhere (0.0.0.0/0) or GitHub Actions IPs
2. Verify the private key in GitHub secrets is correct (copy-paste again)
3. Make sure there are no extra spaces/newlines in the secret

---

## Security Best Practices

### ✅ Do:
- Use a dedicated SSH key for GitHub Actions
- Keep your private key secret (never commit it!)
- Use RSA 4096 or Ed25519 keys
- Rotate keys periodically

### ❌ Don't:
- Use your main SSH key (create a separate one)
- Set a passphrase (GitHub Actions can't use it)
- Commit private keys to git
- Share private keys publicly

---

## Quick Reference

### Get SSH Key:
```bash
cat ~/.ssh/id_rsa
```

### Test SSH:
```bash
ssh -i ~/.ssh/id_rsa ubuntu@YOUR_EC2_IP
```

### Add to EC2:
```bash
ssh-copy-id -i ~/.ssh/id_rsa.pub ubuntu@YOUR_EC2_IP
```

### Add to GitHub:
1. Settings → Secrets → Actions → New secret
2. Name: `EC2_SSH_KEY`
3. Value: Paste private key (entire content)

---

## Example: Complete Setup

```bash
# 1. Generate key (if needed)
ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa_github -N ""

# 2. Add to EC2
ssh-copy-id -i ~/.ssh/id_rsa_github.pub ubuntu@YOUR_EC2_IP

# 3. Test
ssh -i ~/.ssh/id_rsa_github ubuntu@YOUR_EC2_IP

# 4. Get private key for GitHub
cat ~/.ssh/id_rsa_github
# Copy entire output → GitHub Secrets → EC2_SSH_KEY
```

---

**That's it!** Once the SSH key is in GitHub Secrets, your automated deployments will work! 🚀

