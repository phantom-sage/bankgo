# Deployment Guide

This guide covers different deployment strategies for the Bank REST API, from local development to production environments.

## Table of Contents

1. [Docker Deployment](#docker-deployment)
2. [Production Deployment](#production-deployment)
3. [Environment Configuration](#environment-configuration)
4. [Database Setup](#database-setup)
5. [Monitoring and Logging](#monitoring-and-logging)
6. [Security Considerations](#security-considerations)
7. [Backup and Recovery](#backup-and-recovery)

## Docker Deployment

### Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- 4GB RAM minimum
- 10GB disk space

### Local Development

1. **Clone the repository:**
```bash
git clone <repository-url>
cd bank-rest-api
```

2. **Set up environment:**
```bash
make dev-setup
# Edit .env file with your configuration
```

3. **Start all services:**
```bash
make up
# or
docker-compose up -d
```

4. **Verify deployment:**
```bash
make health
# or
curl http://localhost:8080/api/v1/health
```

5. **View logs:**
```bash
make logs
# or
docker-compose logs -f
```

### Available Make Commands

```bash
make help          # Show all available commands
make build         # Build Docker images
make up            # Start development environment
make down          # Stop all services
make logs          # Show service logs
make clean         # Remove all containers and volumes
make test          # Run tests in containers
make db-shell      # Connect to PostgreSQL
make redis-shell   # Connect to Redis
```

### Service URLs

- **API Server**: http://localhost:8080
- **PostgreSQL**: localhost:5432
- **Redis**: localhost:6379
- **Health Check**: http://localhost:8080/api/v1/health

## Production Deployment

### Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- 8GB RAM minimum
- 50GB disk space
- SSL certificate (recommended)
- Domain name (recommended)

### Production Setup

1. **Prepare environment:**
```bash
# Copy production compose file
cp docker-compose.prod.yml docker-compose.yml

# Create production environment file
cp .env.example .env
```

2. **Configure production environment:**
```bash
# Edit .env with production values
nano .env
```

**Critical production settings:**
```bash
# Database Configuration (use strong passwords)
DB_HOST=postgres
DB_PORT=5432
DB_NAME=bankapi
DB_USER=bankuser
DB_PASSWORD=your_very_secure_database_password
DB_SSL_MODE=require
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
DB_CONN_MAX_IDLE_TIME=5m

# Redis Configuration (use strong password)
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=your_very_secure_redis_password
REDIS_DB=0
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONNS=5

# PASETO Authentication (generate secure 32+ character key)
PASETO_SECRET_KEY=your_32_character_production_secret_key
PASETO_EXPIRATION=24h

# Email Configuration (use production SMTP)
SMTP_HOST=smtp.your-provider.com
SMTP_PORT=587
SMTP_USERNAME=your_production_email
SMTP_PASSWORD=your_production_email_password
FROM_EMAIL=noreply@bankapi.com
FROM_NAME=Bank API

# Server Configuration
PORT=8080
HOST=0.0.0.0
GIN_MODE=release
READ_TIMEOUT=30s
WRITE_TIMEOUT=30s
IDLE_TIMEOUT=120s

# Logging Configuration (Zerolog)
LOG_LEVEL=info                    # debug, info, warn, error, fatal
LOG_FORMAT=json                   # json, console
LOG_OUTPUT=both                   # console, file, both
LOG_DIRECTORY=logs               # Directory for log files
LOG_MAX_AGE=30                   # Days to keep log files
LOG_MAX_BACKUPS=10               # Number of backup files to keep
LOG_MAX_SIZE=100                 # Maximum size in MB before rotation
LOG_COMPRESS=true                # Compress rotated files
LOG_LOCAL_TIME=true              # Use local time for file names
LOG_CALLER_INFO=false            # Include caller information
LOG_SAMPLING_ENABLED=false       # Enable log sampling for high volume
LOG_SAMPLING_INITIAL=100         # Initial sampling rate
LOG_SAMPLING_THEREAFTER=100      # Subsequent sampling rate
```

3. **Start production services:**
```bash
make prod-up
# or
docker-compose -f docker-compose.prod.yml up -d
```

4. **Verify deployment:**
```bash
curl https://your-domain.com/api/v1/health
```

### Reverse Proxy Setup (Nginx)

Create `/etc/nginx/sites-available/bankapi`:

```nginx
server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /path/to/your/certificate.crt;
    ssl_certificate_key /path/to/your/private.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains";
}
```

Enable the site:
```bash
sudo ln -s /etc/nginx/sites-available/bankapi /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## Environment Configuration

### Required Environment Variables

| Variable | Production Value | Description |
|----------|------------------|-------------|
| `DB_PASSWORD` | Strong password | Database password |
| `REDIS_PASSWORD` | Strong password | Redis password |
| `PASETO_SECRET_KEY` | 32+ chars | Token signing key |
| `SMTP_PASSWORD` | App password | Email service password |
| `GIN_MODE` | `release` | Gin framework mode |

### Security Environment Variables

```bash
# Generate secure PASETO key (32+ characters)
PASETO_SECRET_KEY=$(openssl rand -base64 32)

# Generate secure database password
DB_PASSWORD=$(openssl rand -base64 24)

# Generate secure Redis password
REDIS_PASSWORD=$(openssl rand -base64 24)
```

### Email Configuration

#### Gmail Setup
1. Enable 2-factor authentication
2. Generate App Password
3. Use App Password in `SMTP_PASSWORD`

```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-16-char-app-password
```

#### SendGrid Setup
```bash
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USERNAME=apikey
SMTP_PASSWORD=your-sendgrid-api-key
```

## Database Setup

### Production Database

#### PostgreSQL Installation (Ubuntu/Debian)

```bash
# Install PostgreSQL
sudo apt update
sudo apt install postgresql-15 postgresql-contrib-15

# Start and enable service
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Create database and user
sudo -u postgres psql
```

```sql
-- Create production database
CREATE DATABASE bankapi_prod;
CREATE USER bankapi_user WITH ENCRYPTED PASSWORD 'your_secure_password';
GRANT ALL PRIVILEGES ON DATABASE bankapi_prod TO bankapi_user;
ALTER USER bankapi_user CREATEDB;
\q
```

#### Database Configuration

```bash
# Edit PostgreSQL configuration
sudo nano /etc/postgresql/15/main/postgresql.conf
```

**Key settings for production:**
```
# Connection settings
max_connections = 100
shared_buffers = 256MB
effective_cache_size = 1GB

# Security
ssl = on
ssl_cert_file = '/path/to/server.crt'
ssl_key_file = '/path/to/server.key'

# Logging
log_statement = 'mod'
log_min_duration_statement = 1000
```

#### Database Migrations

```bash
# Run migrations in production
docker-compose exec bankapi ./main -migrate

# Or manually
psql -h localhost -U bankapi_user -d bankapi_prod -f internal/database/migrations/001_create_users_table.up.sql
psql -h localhost -U bankapi_user -d bankapi_prod -f internal/database/migrations/002_create_accounts_table.up.sql
psql -h localhost -U bankapi_user -d bankapi_prod -f internal/database/migrations/003_create_transfers_table.up.sql
```

### Redis Setup

#### Redis Installation (Ubuntu/Debian)

```bash
# Install Redis
sudo apt update
sudo apt install redis-server

# Configure Redis
sudo nano /etc/redis/redis.conf
```

**Key settings for production:**
```
# Security
requirepass your_secure_redis_password
bind 127.0.0.1

# Memory management
maxmemory 512mb
maxmemory-policy allkeys-lru

# Persistence
save 900 1
save 300 10
save 60 10000
```

```bash
# Start and enable Redis
sudo systemctl start redis-server
sudo systemctl enable redis-server
```

## Monitoring and Logging

### Health Monitoring

Create a health check script:

```bash
#!/bin/bash
# health-check.sh

HEALTH_URL="http://localhost:8080/api/v1/health"
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" $HEALTH_URL)

if [ $RESPONSE -eq 200 ]; then
    echo "$(date): Service is healthy"
    exit 0
else
    echo "$(date): Service is unhealthy (HTTP $RESPONSE)"
    exit 1
fi
```

Add to crontab for regular monitoring:
```bash
# Check every 5 minutes
*/5 * * * * /path/to/health-check.sh >> /var/log/bankapi-health.log 2>&1
```

### Log Management

#### Docker Logs Configuration

```yaml
# In docker-compose.prod.yml
services:
  bankapi:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "5"
```

#### Log Rotation

```bash
# Create logrotate configuration
sudo nano /etc/logrotate.d/bankapi
```

```
/var/log/bankapi/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 root root
    postrotate
        docker-compose restart bankapi
    endscript
}
```

### Metrics Collection

#### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'bankapi'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

## Security Considerations

### Network Security

1. **Firewall Configuration:**
```bash
# Allow only necessary ports
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw deny 5432/tcp   # PostgreSQL (internal only)
sudo ufw deny 6379/tcp   # Redis (internal only)
sudo ufw enable
```

2. **Docker Network Isolation:**
```yaml
# Use custom networks in docker-compose
networks:
  bankapi-network:
    driver: bridge
    internal: true  # No external access
```

### Application Security

1. **Environment Variables:**
   - Never commit `.env` files
   - Use strong, unique passwords
   - Rotate secrets regularly

2. **Database Security:**
   - Use SSL connections
   - Limit database user permissions
   - Regular security updates

3. **API Security:**
   - Enable rate limiting
   - Use HTTPS only
   - Implement proper CORS policies

### SSL/TLS Configuration

1. **Obtain SSL Certificate:**
```bash
# Using Let's Encrypt
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d your-domain.com
```

2. **Auto-renewal:**
```bash
# Add to crontab
0 12 * * * /usr/bin/certbot renew --quiet
```

## Backup and Recovery

### Database Backup

#### Automated Backup Script

```bash
#!/bin/bash
# backup-db.sh

BACKUP_DIR="/backups/bankapi"
DATE=$(date +%Y%m%d_%H%M%S)
DB_NAME="bankapi_prod"
DB_USER="bankapi_user"

# Create backup directory
mkdir -p $BACKUP_DIR

# Create database backup
pg_dump -h localhost -U $DB_USER -d $DB_NAME > $BACKUP_DIR/bankapi_$DATE.sql

# Compress backup
gzip $BACKUP_DIR/bankapi_$DATE.sql

# Remove backups older than 30 days
find $BACKUP_DIR -name "*.sql.gz" -mtime +30 -delete

echo "Backup completed: bankapi_$DATE.sql.gz"
```

#### Schedule Backups

```bash
# Add to crontab (daily at 2 AM)
0 2 * * * /path/to/backup-db.sh >> /var/log/bankapi-backup.log 2>&1
```

### Redis Backup

Redis automatically creates RDB snapshots based on configuration. For additional safety:

```bash
# Manual backup
redis-cli --rdb /backups/redis/dump_$(date +%Y%m%d_%H%M%S).rdb
```

### Recovery Procedures

#### Database Recovery

```bash
# Stop application
docker-compose down

# Restore database
gunzip -c /backups/bankapi/bankapi_YYYYMMDD_HHMMSS.sql.gz | psql -h localhost -U bankapi_user -d bankapi_prod

# Start application
docker-compose up -d
```

#### Redis Recovery

```bash
# Stop Redis
docker-compose stop redis

# Replace RDB file
cp /backups/redis/dump_YYYYMMDD_HHMMSS.rdb /var/lib/redis/dump.rdb

# Start Redis
docker-compose start redis
```

## Troubleshooting

### Common Issues

1. **Database Connection Failed:**
```bash
# Check PostgreSQL status
sudo systemctl status postgresql

# Check connection
psql -h localhost -U bankapi_user -d bankapi_prod

# Check logs
sudo journalctl -u postgresql
```

2. **Redis Connection Failed:**
```bash
# Check Redis status
sudo systemctl status redis-server

# Test connection
redis-cli -a your_redis_password ping

# Check logs
sudo journalctl -u redis-server
```

3. **Email Sending Failed:**
```bash
# Test SMTP connection
telnet smtp.gmail.com 587

# Check application logs
docker-compose logs worker
```

4. **High Memory Usage:**
```bash
# Check container resource usage
docker stats

# Adjust resource limits in docker-compose.yml
deploy:
  resources:
    limits:
      memory: 512M
```

### Performance Optimization

1. **Database Optimization:**
```sql
-- Add indexes for frequently queried columns
CREATE INDEX CONCURRENTLY idx_accounts_user_currency ON accounts(user_id, currency);
CREATE INDEX CONCURRENTLY idx_transfers_created_at ON transfers(created_at DESC);

-- Analyze query performance
EXPLAIN ANALYZE SELECT * FROM accounts WHERE user_id = 1;
```

2. **Redis Optimization:**
```
# Optimize memory usage
maxmemory-policy allkeys-lru
maxmemory 256mb

# Enable compression
rdbcompression yes
```

3. **Application Optimization:**
```bash
# Enable Go profiling
GIN_MODE=release
GOMAXPROCS=2  # Match container CPU limit
```

## Maintenance

### Regular Maintenance Tasks

1. **Weekly:**
   - Review application logs
   - Check disk space usage
   - Verify backup integrity

2. **Monthly:**
   - Update dependencies
   - Review security logs
   - Performance analysis

3. **Quarterly:**
   - Security audit
   - Disaster recovery testing
   - Capacity planning review

### Update Procedures

1. **Application Updates:**
```bash
# Pull latest code
git pull origin main

# Rebuild and deploy
docker-compose build
docker-compose up -d
```

2. **Database Updates:**
```bash
# Run new migrations
docker-compose exec bankapi ./main -migrate
```

3. **Security Updates:**
```bash
# Update base images
docker-compose pull
docker-compose up -d
```