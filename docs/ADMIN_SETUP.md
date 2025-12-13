# Super Admin Setup Guide

This guide explains how to create the first super admin account for Code-Sherlock.

## Prerequisites

- Database is set up and running
- Backend dependencies are installed (`make deps`)

## Creating a Super Admin

### Option 1: Using Make (Recommended)

```bash
make create-admin \
  EMAIL=admin@example.com \
  PASSWORD=your-secure-password \
  NAME="Super Admin" \
  DB_URL="postgres://user:password@localhost/sherlock?sslmode=disable"
```

### Option 2: Using Go Run Directly

```bash
cd backend
go run ./cmd/create-admin/main.go \
  -email admin@example.com \
  -password your-secure-password \
  -name "Super Admin" \
  -db "postgres://user:password@localhost/sherlock?sslmode=disable"
```

### Option 3: Using Compiled Binary

First, build the binary:
```bash
make build
```

Then run it:
```bash
./bin/create-admin \
  -email admin@example.com \
  -password your-secure-password \
  -name "Super Admin" \
  -db "postgres://user:password@localhost/sherlock?sslmode=disable"
```

## Database URL Format

The database URL should follow PostgreSQL connection string format:
```
postgres://[user]:[password]@[host]:[port]/[database]?[parameters]
```

Examples:
- Local: `postgres://postgres:postgres@localhost:5432/sherlock?sslmode=disable`
- Docker: `postgres://postgres:postgres@db:5432/sherlock?sslmode=disable`
- Production: `postgres://user:pass@host.example.com:5432/sherlock?sslmode=require`

## After Creating Super Admin

1. Start the server:
   ```bash
   make run
   ```

2. Navigate to the login page: `http://localhost:8080/login`

3. Log in with the super admin credentials you just created

4. You'll be redirected to `/admin` dashboard where you can see:
   - System-wide statistics
   - All organizations
   - All users

## Security Notes

- Choose a strong password for the super admin account
- Keep super admin credentials secure
- Consider using environment variables for database URL in production
- Super admin accounts have full system access - use sparingly

## Troubleshooting

### "User already exists"
If you see this error, the email is already registered. You can either:
- Use a different email
- Manually update the user's role in the database:
  ```sql
  UPDATE users SET role = 'super_admin' WHERE email = 'admin@example.com';
  ```

### "Failed to connect to database"
- Verify database is running
- Check database URL format
- Ensure database credentials are correct
- Check network connectivity if using remote database
