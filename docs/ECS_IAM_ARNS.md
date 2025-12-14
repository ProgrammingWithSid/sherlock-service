# ECS IAM ARNs Configuration Guide

## Overview

This guide provides the ARNs you need to specify for ECS resources in your IAM policy to restrict access to your Code-Sherlock ECS resources.

## Your ECS Resources

Based on your configuration:
- **Cluster**: `sherlock-cluster`
- **Services**:
  - `sherlock-server`
  - `sherlock-worker`
- **Task Definitions**:
  - `sherlock-server`
  - `sherlock-worker`

---

## ARN Formats

Replace the following placeholders:
- `REGION`: Your AWS region (e.g., `us-east-1`, `ap-south-1`)
- `ACCOUNT_ID`: Your AWS account ID (12-digit number)

---

## Specific ARNs to Add

### 1. Cluster ARN

```
arn:aws:ecs:REGION:ACCOUNT_ID:cluster/sherlock-cluster
```

**Actions affected**: CreateCluster, DescribeClusters, UpdateCluster, DeleteCluster, and 17 more actions

---

### 2. Service ARNs

**Server Service:**
```
arn:aws:ecs:REGION:ACCOUNT_ID:service/sherlock-cluster/sherlock-server
```

**Worker Service:**
```
arn:aws:ecs:REGION:ACCOUNT_ID:service/sherlock-cluster/sherlock-worker
```

**Actions affected**: CreateService, UpdateService, DescribeServices, DeleteService, and 15 more actions

**Note**: Add both service ARNs if you need access to both services.

---

### 3. Task Definition ARNs

**Server Task Definition:**
```
arn:aws:ecs:REGION:ACCOUNT_ID:task-definition/sherlock-server:*
```

**Worker Task Definition:**
```
arn:aws:ecs:REGION:ACCOUNT_ID:task-definition/sherlock-worker:*
```

**Actions affected**: RegisterTaskDefinition, DescribeTaskDefinition, DeregisterTaskDefinition, and 6 more actions

**Note**: Use `:*` to allow all revisions, or specify specific revision numbers like `:1`, `:2`, etc.

---

### 4. Task ARNs (Dynamic)

**Pattern for all tasks:**
```
arn:aws:ecs:REGION:ACCOUNT_ID:task/sherlock-cluster/*
```

**Actions affected**: DescribeTasks, StopTask, StartTask, RunTask, and 7 more actions

**Note**: Tasks are created dynamically, so use wildcard `/*` to allow access to all tasks in the cluster.

---

### 5. Container Instance ARNs (Fargate - Not Applicable)

**If using EC2 launch type:**
```
arn:aws:ecs:REGION:ACCOUNT_ID:container-instance/sherlock-cluster/*
```

**Note**: You're using Fargate, so container instances are not applicable. Skip this resource type.

---

### 6. Capacity Provider ARNs (If Using)

**If you have custom capacity providers:**
```
arn:aws:ecs:REGION:ACCOUNT_ID:capacity-provider/YOUR_CAPACITY_PROVIDER_NAME
```

**Note**: Fargate uses AWS-managed capacity providers. Only add if you've created custom capacity providers.

---

### 7. Task Set ARNs (If Using Blue/Green Deployments)

**Pattern for task sets:**
```
arn:aws:ecs:REGION:ACCOUNT_ID:task-set/sherlock-cluster/sherlock-server/*
arn:aws:ecs:REGION:ACCOUNT_ID:task-set/sherlock-cluster/sherlock-worker/*
```

**Actions affected**: CreateTaskSet, DescribeTaskSets, UpdateTaskSet, DeleteTaskSet, and 6 more actions

**Note**: Only needed if using blue/green deployments with CodeDeploy.

---

## Recommended IAM Policy (Minimal)

For GitHub Actions deployment, you typically need:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecs:DescribeClusters",
        "ecs:DescribeServices",
        "ecs:DescribeTaskDefinition",
        "ecs:RegisterTaskDefinition",
        "ecs:UpdateService",
        "ecs:DescribeTasks"
      ],
      "Resource": [
        "arn:aws:ecs:REGION:ACCOUNT_ID:cluster/sherlock-cluster",
        "arn:aws:ecs:REGION:ACCOUNT_ID:service/sherlock-cluster/sherlock-server",
        "arn:aws:ecs:REGION:ACCOUNT_ID:service/sherlock-cluster/sherlock-worker",
        "arn:aws:ecs:REGION:ACCOUNT_ID:task-definition/sherlock-server:*",
        "arn:aws:ecs:REGION:ACCOUNT_ID:task-definition/sherlock-worker:*",
        "arn:aws:ecs:REGION:ACCOUNT_ID:task/sherlock-cluster/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "ecs:ListClusters",
        "ecs:ListServices",
        "ecs:ListTaskDefinitions"
      ],
      "Resource": "*"
    }
  ]
}
```

---

## Quick Reference: ARN Template

Replace `REGION` and `ACCOUNT_ID` in these templates:

```
Cluster:        arn:aws:ecs:REGION:ACCOUNT_ID:cluster/sherlock-cluster
Server Service: arn:aws:ecs:REGION:ACCOUNT_ID:service/sherlock-cluster/sherlock-server
Worker Service: arn:aws:ecs:REGION:ACCOUNT_ID:service/sherlock-cluster/sherlock-worker
Server Tasks:   arn:aws:ecs:REGION:ACCOUNT_ID:task-definition/sherlock-server:*
Worker Tasks:   arn:aws:ecs:REGION:ACCOUNT_ID:task-definition/sherlock-worker:*
All Tasks:      arn:aws:ecs:REGION:ACCOUNT_ID:task/sherlock-cluster/*
```

---

## Finding Your Account ID and Region

**Account ID:**
```bash
aws sts get-caller-identity --query Account --output text
```

**Region:**
Check your AWS CLI configuration:
```bash
aws configure get region
```

Or check your ECS cluster:
```bash
aws ecs describe-clusters --clusters sherlock-cluster --query 'clusters[0].clusterArn'
```

---

## Security Best Practices

1. **Use Specific ARNs**: Instead of `*`, specify exact resource ARNs
2. **Least Privilege**: Only grant permissions needed for deployment
3. **Separate Policies**: Create separate policies for different roles (deployment vs. monitoring)
4. **Review Regularly**: Periodically review and audit IAM policies

---

## Common Issues

### Issue: "Access Denied" when deploying
**Solution**: Ensure all required ARNs are added, especially task definitions with `:*` wildcard

### Issue: Can't see tasks
**Solution**: Add task ARN pattern: `arn:aws:ecs:REGION:ACCOUNT_ID:task/sherlock-cluster/*`

### Issue: Can't update service
**Solution**: Ensure service ARNs are correctly formatted with cluster name prefix

---

**Last Updated**: 2024-12-13
**Cluster Name**: `sherlock-cluster`
**Services**: `sherlock-server`, `sherlock-worker`
