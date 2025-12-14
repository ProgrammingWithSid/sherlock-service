# Local Testing Guide

## Testing Feedback API Locally

### 1. Start the Backend Server

```bash
cd backend
go run ./cmd/server
```

The server will start on port 3000 (default) and automatically create the `review_feedback` table on startup.

### 2. Test the Feedback API

**Fix the Authorization header** - Remove duplicate "Bearer":

```bash
# ❌ WRONG (has duplicate Bearer)
curl -H "Authorization: Bearer Bearer TOKEN" ...

# ✅ CORRECT
curl -H "Authorization: Bearer TOKEN" ...
```

**Test GetFeedbackPatterns:**

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
     -H "X-Org-ID: YOUR_ORG_ID" \
     http://localhost:3000/api/v1/feedback/patterns
```

**Test GetTeamPreferences:**

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
     -H "X-Org-ID: YOUR_ORG_ID" \
     http://localhost:3000/api/v1/feedback/preferences
```

### 3. Use the Test Script

```bash
./scripts/test-feedback-local.sh YOUR_TOKEN YOUR_ORG_ID
```

### Expected Responses

**When no feedback exists yet:**
```json
{
  "feedback_distribution": {},
  "total_feedback": 0
}
```

**When feedback exists:**
```json
{
  "feedback_distribution": {
    "accepted": 10,
    "dismissed": 5,
    "fixed": 3
  },
  "total_feedback": 18,
  "acceptance_rate": 55.56
}
```

## Troubleshooting

### 404 Not Found
- **Cause**: Server not restarted with new code
- **Fix**: Restart the server (`Ctrl+C` then `go run ./cmd/server`)

### 401 Unauthorized
- **Cause**: Missing or invalid token/org ID
- **Fix**: Check your token and org ID are correct

### 500 Internal Server Error
- **Cause**: Database table doesn't exist
- **Fix**: Restart server - table is created automatically on startup

### Database Connection Error
- **Cause**: PostgreSQL not running or wrong DATABASE_URL
- **Fix**: Check `DATABASE_URL` environment variable and PostgreSQL status

## Route Structure

- Base: `/api/v1`
- Feedback routes: `/api/v1/feedback/*`
  - `GET /api/v1/feedback/patterns` - Get feedback patterns
  - `GET /api/v1/feedback/preferences` - Get team preferences
  - `POST /api/v1/feedback` - Record feedback

All feedback routes require:
- `Authorization: Bearer <token>` header
- `X-Org-ID: <org_id>` header

