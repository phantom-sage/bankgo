# Troubleshooting Guide

This guide helps diagnose and resolve common issues with the Bank REST API.

## Table of Contents

1. [Quick Diagnostics](#quick-diagnostics)
2. [Database Issues](#database-issues)
3. [Redis Issues](#redis-issues)
4. [Authentication Issues](#authentication-issues)
5. [Email Issues](#email-issues)
6. [Performance Issues](#performance-issues)
7. [Docker Issues](#docker-issues)
8. [API Issues](#api-issues)
9. [Logging and Debugging](#logging-and-debugging)

## Quick Diagnostics

### Health Check

First, check the overall service health:

```bash
curl http://localhost:8080/api/v1/health
```

**Expected Response:**
```json
{
  "status": "healthy",
  "version": "v1.0.0",
  "timestamp": "2024-01-15T15:30:00Z",
  "services": {
    "database": {
      "status": "connected",
      "response_time": "2ms"
    },
    "redis": {
      "status": "connected",
      "response_time": "1ms"
    }
  }
}
```

### Service Status Check

```bash
# Check all Docker containers
docker-compose ps

# Check individual service logs
docker-compose logs bankapi
docker-compose logs postgres
docker-compose logs redis
docker-compose logs worker
```

### Resource Usage

```bash
# Check container resource usage
docker stats

# Check system resources
free -h
df -h
```

## Database Issues

### Connection Failed

**Symptoms:**
- API returns 500 errors
- Health check shows database disconnected
- "connection refused" errors in logs

**Diagnosis:**
```bash
# Check PostgreSQL container status
docker-compose ps postgres

# Check PostgreSQL logs
docker-compose logs postgres

# Test direct connection
docker-compose exec postgres psql -U bankuser -d bankapi
```

**Solutions:**

1. **Container not running:**
```bash
docker-compose up -d postgres
```

2. **Wrong credentials:**
```bash
# Check environment variables
docker-compose exec bankapi env | grep DB_
```

3. **Database doesn't exist:**
```bash
# Create database
docker-compose exec postgres createdb -U bankuser bankapi
```

4. **Connection timeout:**
```bash
# Check network connectivity
docker-compose exec bankapi ping postgres
```

### Migration Errors

**Symptoms:**
- Tables don't exist
- Schema version mismatch
- Migration failed errors

**Diagnosis:**
```bash
# Check if tables exist
docker-compose exec postgres psql -U bankuser -d bankapi -c "\dt"

# Check migration status
docker-compose logs bankapi | grep migration
```

**Solutions:**

1. **Run migrations manually:**
```bash
# Connect to database
docker-compose exec postgres psql -U bankuser -d bankapi

# Run migration files
\i /docker-entrypoint-initdb.d/001_create_users_table.up.sql
\i /docker-entrypoint-initdb.d/002_create_accounts_table.up.sql
\i /docker-entrypoint-initdb.d/003_create_transfers_table.up.sql
```

2. **Reset database (development only):**
```bash
docker-compose down -v
docker-compose up -d
```

### Performance Issues

**Symptoms:**
- Slow query responses
- Database timeouts
- High CPU usage

**Diagnosis:**
```bash
# Check active connections
docker-compose exec postgres psql -U bankuser -d bankapi -c "SELECT count(*) FROM pg_stat_activity;"

# Check slow queries
docker-compose exec postgres psql -U bankuser -d bankapi -c "SELECT query, mean_time, calls FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"
```

**Solutions:**

1. **Add missing indexes:**
```sql
CREATE INDEX CONCURRENTLY idx_accounts_user_id ON accounts(user_id);
CREATE INDEX CONCURRENTLY idx_transfers_from_account ON transfers(from_account_id);
CREATE INDEX CONCURRENTLY idx_transfers_to_account ON transfers(to_account_id);
```

2. **Optimize connection pool:**
```bash
# In .env file
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
```

## Redis Issues

### Connection Failed

**Symptoms:**
- Email jobs not processing
- Health check shows Redis disconnected
- "connection refused" errors

**Diagnosis:**
```bash
# Check Redis container status
docker-compose ps redis

# Check Redis logs
docker-compose logs redis

# Test Redis connection
docker-compose exec redis redis-cli ping
```

**Solutions:**

1. **Container not running:**
```bash
docker-compose up -d redis
```

2. **Authentication failed:**
```bash
# Test with password
docker-compose exec redis redis-cli -a redispass123 ping

# Check environment variables
docker-compose exec worker env | grep REDIS_
```

3. **Memory issues:**
```bash
# Check Redis memory usage
docker-compose exec redis redis-cli info memory

# Clear Redis data (development only)
docker-compose exec redis redis-cli FLUSHALL
```

### Queue Processing Issues

**Symptoms:**
- Welcome emails not sent
- Jobs stuck in queue
- Worker not processing tasks

**Diagnosis:**
```bash
# Check worker logs
docker-compose logs worker

# Check queue status
docker-compose exec redis redis-cli -a redispass123 LLEN asynq:default

# List failed jobs
docker-compose exec redis redis-cli -a redispass123 LLEN asynq:default:failed
```

**Solutions:**

1. **Restart worker:**
```bash
docker-compose restart worker
```

2. **Clear failed jobs:**
```bash
docker-compose exec redis redis-cli -a redispass123 DEL asynq:default:failed
```

3. **Check email configuration:**
```bash
# Verify SMTP settings
docker-compose exec worker env | grep SMTP_
```

## Authentication Issues

### Token Validation Failed

**Symptoms:**
- 401 Unauthorized responses
- "invalid token" errors
- Users can't access protected endpoints

**Diagnosis:**
```bash
# Check PASETO configuration
docker-compose exec bankapi env | grep PASETO_

# Check token in request
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/accounts
```

**Solutions:**

1. **Invalid PASETO secret:**
```bash
# Ensure secret is at least 32 characters
PASETO_SECRET_KEY=your_32_character_secret_key_here
```

2. **Token expired:**
```bash
# Login again to get new token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}'
```

3. **Malformed Authorization header:**
```bash
# Correct format
Authorization: Bearer v2.local.xxx...

# Incorrect format (missing Bearer)
Authorization: v2.local.xxx...
```

### Login Failed

**Symptoms:**
- 401 responses on login
- "invalid credentials" errors
- Password validation failures

**Diagnosis:**
```bash
# Check user exists in database
docker-compose exec postgres psql -U bankuser -d bankapi -c "SELECT id, email FROM users WHERE email = 'user@example.com';"

# Check password hash
docker-compose exec postgres psql -U bankuser -d bankapi -c "SELECT password_hash FROM users WHERE email = 'user@example.com';"
```

**Solutions:**

1. **User doesn't exist:**
```bash
# Register user first
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password","first_name":"John","last_name":"Doe"}'
```

2. **Wrong password:**
```bash
# Use correct password or reset (implementation dependent)
```

## Email Issues

### Welcome Emails Not Sent

**Symptoms:**
- Users not receiving welcome emails
- Email jobs failing
- SMTP connection errors

**Diagnosis:**
```bash
# Check worker logs
docker-compose logs worker

# Check email queue
docker-compose exec redis redis-cli -a redispass123 LLEN asynq:default

# Test SMTP connection
telnet smtp.gmail.com 587
```

**Solutions:**

1. **SMTP configuration issues:**
```bash
# For Gmail, use App Password
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-16-char-app-password  # Not regular password
```

2. **Firewall blocking SMTP:**
```bash
# Check if port 587 is accessible
nc -zv smtp.gmail.com 587
```

3. **Email provider restrictions:**
```bash
# Check provider-specific requirements
# Gmail: Enable 2FA and use App Password
# SendGrid: Use API key as password
```

### Email Queue Stuck

**Symptoms:**
- Jobs accumulating in queue
- No email processing
- Worker errors

**Diagnosis:**
```bash
# Check queue length
docker-compose exec redis redis-cli -a redispass123 LLEN asynq:default

# Check failed jobs
docker-compose exec redis redis-cli -a redispass123 LLEN asynq:default:failed

# Check worker status
docker-compose ps worker
```

**Solutions:**

1. **Restart worker:**
```bash
docker-compose restart worker
```

2. **Clear queue (development only):**
```bash
docker-compose exec redis redis-cli -a redispass123 FLUSHALL
```

3. **Check worker configuration:**
```bash
# Ensure worker has correct environment variables
docker-compose exec worker env | grep -E "(SMTP_|REDIS_)"
```

## Performance Issues

### Slow API Responses

**Symptoms:**
- High response times
- Timeouts
- Poor user experience

**Diagnosis:**
```bash
# Check response times
curl -w "@curl-format.txt" -o /dev/null -s http://localhost:8080/api/v1/health

# Create curl-format.txt:
echo "     time_namelookup:  %{time_namelookup}\n
        time_connect:  %{time_connect}\n
     time_appconnect:  %{time_appconnect}\n
    time_pretransfer:  %{time_pretransfer}\n
       time_redirect:  %{time_redirect}\n
  time_starttransfer:  %{time_starttransfer}\n
                     ----------\n
          time_total:  %{time_total}\n" > curl-format.txt
```

**Solutions:**

1. **Database optimization:**
```sql
-- Add missing indexes
CREATE INDEX CONCURRENTLY idx_accounts_user_id ON accounts(user_id);

-- Analyze queries
EXPLAIN ANALYZE SELECT * FROM accounts WHERE user_id = 1;
```

2. **Connection pool tuning:**
```bash
# Optimize database connections
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
```

3. **Resource limits:**
```yaml
# In docker-compose.yml
deploy:
  resources:
    limits:
      memory: 1G
      cpus: '1.0'
```

### High Memory Usage

**Symptoms:**
- Container OOM kills
- System slowdown
- Memory warnings

**Diagnosis:**
```bash
# Check container memory usage
docker stats

# Check system memory
free -h

# Check Go memory usage (if accessible)
curl http://localhost:8080/debug/pprof/heap
```

**Solutions:**

1. **Increase container limits:**
```yaml
deploy:
  resources:
    limits:
      memory: 1G
```

2. **Optimize Go garbage collection:**
```bash
# Set GOGC environment variable
GOGC=100  # Default, lower values = more frequent GC
```

3. **Check for memory leaks:**
```bash
# Monitor memory usage over time
watch docker stats
```

## Docker Issues

### Container Won't Start

**Symptoms:**
- Container exits immediately
- "Exited (1)" status
- Build failures

**Diagnosis:**
```bash
# Check container status
docker-compose ps

# Check container logs
docker-compose logs <service-name>

# Check Docker daemon
sudo systemctl status docker
```

**Solutions:**

1. **Build issues:**
```bash
# Rebuild images
docker-compose build --no-cache

# Check Dockerfile syntax
docker build -t test .
```

2. **Port conflicts:**
```bash
# Check if ports are in use
netstat -tulpn | grep :8080

# Change ports in docker-compose.yml
ports:
  - "8081:8080"  # Use different host port
```

3. **Volume issues:**
```bash
# Remove volumes and recreate
docker-compose down -v
docker-compose up -d
```

### Build Failures

**Symptoms:**
- Docker build errors
- Missing dependencies
- Go build failures

**Diagnosis:**
```bash
# Build with verbose output
docker-compose build --progress=plain

# Check Go modules
go mod verify
go mod tidy
```

**Solutions:**

1. **Go module issues:**
```bash
# Clean module cache
go clean -modcache
go mod download
```

2. **Docker cache issues:**
```bash
# Build without cache
docker-compose build --no-cache
```

3. **Base image issues:**
```bash
# Update base images
docker pull golang:1.24.3-alpine
docker pull alpine:latest
```

## API Issues

### 500 Internal Server Error

**Symptoms:**
- Unexpected server errors
- Generic error responses
- Application crashes

**Diagnosis:**
```bash
# Check application logs
docker-compose logs bankapi

# Check error patterns
docker-compose logs bankapi | grep ERROR

# Test specific endpoints
curl -v http://localhost:8080/api/v1/health
```

**Solutions:**

1. **Database connection issues:**
```bash
# Verify database connectivity
docker-compose exec bankapi ping postgres
```

2. **Configuration errors:**
```bash
# Check environment variables
docker-compose exec bankapi env
```

3. **Application bugs:**
```bash
# Enable debug mode (development only)
GIN_MODE=debug
```

### Rate Limiting Issues

**Symptoms:**
- 429 Too Many Requests
- Legitimate requests blocked
- Rate limit too restrictive

**Diagnosis:**
```bash
# Check rate limit headers
curl -I http://localhost:8080/api/v1/health

# Check rate limit configuration
docker-compose exec bankapi env | grep RATE_
```

**Solutions:**

1. **Adjust rate limits:**
```bash
# In application configuration
RATE_LIMIT_PER_MINUTE=100
RATE_LIMIT_PER_HOUR=1000
```

2. **Clear rate limit cache:**
```bash
# Clear Redis rate limit data
docker-compose exec redis redis-cli -a redispass123 FLUSHDB
```

## Logging and Debugging

### Enable Debug Logging

**Development:**
```bash
# Set debug mode
GIN_MODE=debug

# Restart application
docker-compose restart bankapi
```

**Production:**
```bash
# Enable structured logging
LOG_LEVEL=debug
LOG_FORMAT=json

# Restart application
docker-compose restart bankapi
```

### Log Analysis

**Common log patterns:**
```bash
# Database errors
docker-compose logs bankapi | grep "database"

# Authentication errors
docker-compose logs bankapi | grep "auth"

# Email errors
docker-compose logs worker | grep "email"

# Performance issues
docker-compose logs bankapi | grep "slow"
```

### Debug Tools

**Database debugging:**
```bash
# Connect to database
docker-compose exec postgres psql -U bankuser -d bankapi

# Check active queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query 
FROM pg_stat_activity 
WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes';
```

**Redis debugging:**
```bash
# Connect to Redis
docker-compose exec redis redis-cli -a redispass123

# Monitor commands
MONITOR

# Check memory usage
INFO memory
```

### Performance Profiling

**Go profiling (development):**
```bash
# Enable pprof endpoint
go tool pprof http://localhost:8080/debug/pprof/profile

# Memory profiling
go tool pprof http://localhost:8080/debug/pprof/heap

# CPU profiling
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
```

## Getting Help

### Information to Collect

When reporting issues, include:

1. **System information:**
```bash
docker --version
docker-compose --version
uname -a
```

2. **Service status:**
```bash
docker-compose ps
docker-compose logs --tail=50
```

3. **Configuration:**
```bash
# Sanitized environment variables (remove passwords)
docker-compose exec bankapi env | grep -v PASSWORD
```

4. **Error reproduction:**
```bash
# Exact commands that cause the issue
# Expected vs actual behavior
# Error messages and logs
```

### Support Channels

- **Documentation**: Check README.md and docs/ directory
- **Issues**: Create GitHub issue with collected information
- **Logs**: Always include relevant log excerpts
- **Environment**: Specify development vs production

### Emergency Procedures

**Service down:**
```bash
# Quick restart
docker-compose restart

# Full reset (development only)
docker-compose down -v
docker-compose up -d
```

**Data corruption:**
```bash
# Restore from backup
# See DEPLOYMENT.md for backup/restore procedures
```

**Security incident:**
```bash
# Rotate secrets immediately
# Check logs for suspicious activity
# Update passwords and tokens
```