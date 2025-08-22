# Logging Best Practices Guide

## Overview

This guide provides best practices for using the zerolog-based logging system in the Bank REST API. The logging system is designed for high performance, comprehensive monitoring, and operational excellence.

## Quick Start

### Basic Configuration

```bash
# Development (human-readable logs)
LOG_LEVEL=debug
LOG_FORMAT=console
LOG_OUTPUT=console

# Production (structured JSON logs)
LOG_LEVEL=info
LOG_FORMAT=json
LOG_OUTPUT=both
LOG_DIRECTORY=logs
LOG_COMPRESS=true
```

### Viewing Logs

```bash
# Real-time log viewing
tail -f logs/app-$(date +%Y-%m-%d).log

# Structured log analysis with jq
tail -f logs/app-$(date +%Y-%m-%d).log | jq .

# Docker container logs
docker-compose logs -f bankapi
```

## Log Levels and When to Use Them

### Debug Level
**Use for:** Detailed debugging information, variable values, flow control

```go
logger.Debug().
    Str("user_id", userID).
    Str("operation", "validate_account").
    Msg("Starting account validation")
```

**When to enable:** Development, troubleshooting specific issues
**Performance impact:** High - only use when needed

### Info Level
**Use for:** Normal application flow, successful operations, business events

```go
logger.Info().
    Int64("user_id", user.ID).
    Str("account_id", account.ID).
    Str("currency", account.Currency).
    Msg("Account created successfully")
```

**When to enable:** Always in production
**Performance impact:** Low

### Warn Level
**Use for:** Potentially harmful situations, deprecated features, recoverable errors

```go
logger.Warn().
    Str("user_id", userID).
    Int("retry_count", retryCount).
    Msg("Email delivery failed, will retry")
```

**When to enable:** Always
**Performance impact:** Very low

### Error Level
**Use for:** Error conditions that don't stop the application

```go
logger.Error().
    Err(err).
    Str("user_id", userID).
    Str("operation", "transfer").
    Msg("Transfer failed due to insufficient funds")
```

**When to enable:** Always
**Performance impact:** Minimal

### Fatal Level
**Use for:** Critical errors that cause application shutdown

```go
logger.Fatal().
    Err(err).
    Msg("Failed to connect to database")
```

**When to enable:** Always
**Performance impact:** None (application exits)

## Structured Logging Best Practices

### Field Naming Conventions

Use consistent field names across the application:

```go
// Good - consistent naming
logger.Info().
    Str("request_id", requestID).
    Int64("user_id", userID).
    Str("user_email", user.Email).
    Dur("duration", elapsed).
    Int("status_code", 200).
    Msg("Request completed")

// Avoid - inconsistent naming
logger.Info().
    Str("reqId", requestID).        // inconsistent
    Int64("userId", userID).        // inconsistent
    Str("email", user.Email).       // too generic
    Msg("Done")                     // not descriptive
```

### Standard Fields

Use these standard fields consistently:

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `request_id` | string | Unique request identifier | `20240115103045-abc12345` |
| `user_id` | int64 | User identifier | `123` |
| `user_email` | string | User email address | `user@example.com` |
| `duration` | duration | Operation duration | `45ms` |
| `status_code` | int | HTTP status code | `200` |
| `method` | string | HTTP method | `POST` |
| `path` | string | Request path | `/api/v1/transfers` |
| `client_ip` | string | Client IP address | `192.168.1.100` |
| `error_type` | string | Error category | `validation_error` |
| `component` | string | System component | `database`, `email`, `auth` |

### Contextual Logging

Always include relevant context:

```go
// Good - includes context
logger.Info().
    Str("request_id", ctx.Value("request_id").(string)).
    Int64("user_id", user.ID).
    Str("operation", "create_account").
    Str("currency", req.Currency).
    Msg("Account creation started")

// Poor - missing context
logger.Info().Msg("Creating account")
```

### Error Logging

Include comprehensive error information:

```go
// Good - comprehensive error logging
logger.Error().
    Err(err).
    Str("request_id", requestID).
    Int64("user_id", userID).
    Str("operation", "transfer_money").
    Str("from_account", fromAccount).
    Str("to_account", toAccount).
    Str("amount", amount.String()).
    Str("error_type", "insufficient_funds").
    Msg("Money transfer failed")

// Poor - minimal error information
logger.Error().Err(err).Msg("Transfer failed")
```

## Performance Optimization

### Log Sampling

For high-volume scenarios, use sampling to reduce log volume:

```bash
# Enable sampling - log every 10th message initially, then every 100th
LOG_SAMPLING_ENABLED=true
LOG_SAMPLING_INITIAL=10
LOG_SAMPLING_THEREAFTER=100
```

### Conditional Logging

Use zerolog's conditional logging for expensive operations:

```go
// Good - only evaluates expensive operation if debug is enabled
if logger.Debug().Enabled() {
    expensiveData := generateExpensiveDebugData()
    logger.Debug().
        Interface("debug_data", expensiveData).
        Msg("Debug information")
}

// Poor - always evaluates expensive operation
logger.Debug().
    Interface("debug_data", generateExpensiveDebugData()).
    Msg("Debug information")
```

### Lazy Evaluation

Use zerolog's lazy evaluation for better performance:

```go
// Good - lazy evaluation
logger.Info().
    Str("user_id", userID).
    Func(func(e *zerolog.Event) {
        if complexCondition {
            e.Str("complex_field", expensiveCalculation())
        }
    }).
    Msg("Operation completed")
```

## Security Considerations

### Sensitive Data Protection

Never log sensitive information:

```go
// Good - sensitive data excluded
logger.Info().
    Str("user_email", user.Email).
    Str("operation", "login").
    Msg("User authentication successful")

// NEVER DO - exposes sensitive data
logger.Info().
    Str("password", password).        // NEVER
    Str("token", authToken).          // NEVER
    Str("credit_card", cardNumber).   // NEVER
    Msg("User data")
```

### Data Sanitization

Sanitize data before logging:

```go
func sanitizeEmail(email string) string {
    parts := strings.Split(email, "@")
    if len(parts) != 2 {
        return "[invalid_email]"
    }
    return parts[0][:1] + "***@" + parts[1]
}

logger.Info().
    Str("user_email_sanitized", sanitizeEmail(user.Email)).
    Msg("User operation")
```

## Audit Logging

### Security Events

Log all security-relevant events:

```go
// Authentication events
auditLogger.LogAuthentication(user.ID, user.Email, "login", "success")

// Account operations
auditLogger.LogAccountOperation(user.ID, account.ID, "create", "success")

// Money transfers
auditLogger.LogTransfer(fromAccount, toAccount, amount, "success")

// Administrative actions
auditLogger.LogAdminAction(admin.ID, "delete_user", user.ID, "success")
```

### Compliance Requirements

Ensure audit logs meet compliance requirements:

- **Immutability**: Use append-only log files
- **Integrity**: Consider log signing for critical environments
- **Retention**: Configure appropriate retention periods
- **Access Control**: Restrict access to audit logs

## File Management

### Rotation Strategy

Configure appropriate rotation settings:

```bash
# Daily rotation with 30-day retention
LOG_MAX_AGE=30
LOG_MAX_BACKUPS=30
LOG_COMPRESS=true

# Size-based rotation for high-volume systems
LOG_MAX_SIZE=100  # 100MB per file
LOG_MAX_BACKUPS=50
```

### Disk Space Management

Monitor and manage disk space:

```bash
# Check log directory size
du -sh logs/

# Clean up old logs manually if needed
find logs/ -name "*.log.gz" -mtime +30 -delete

# Monitor disk space
df -h
```

### Backup and Archival

For production systems:

```bash
# Daily backup of log files
0 2 * * * tar -czf /backup/logs-$(date +%Y%m%d).tar.gz logs/

# Archive to long-term storage
0 3 * * 0 aws s3 cp /backup/logs-*.tar.gz s3://your-log-archive/
```

## Monitoring and Alerting

### Log-Based Metrics

Create alerts based on log patterns:

```bash
# Error rate monitoring
grep -c "level.*error" logs/app-$(date +%Y-%m-%d).log

# Performance monitoring
grep "duration_ms" logs/app-$(date +%Y-%m-%d).log | jq '.duration_ms' | awk '{sum+=$1; count++} END {print sum/count}'

# Security monitoring
grep "authentication.*failed" logs/app-$(date +%Y-%m-%d).log
```

### Health Checks

Monitor logging system health:

```go
func (lm *LoggerManager) HealthCheck() error {
    // Check log directory writability
    testFile := filepath.Join(lm.config.Directory, ".health_check")
    if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
        return fmt.Errorf("log directory not writable: %w", err)
    }
    os.Remove(testFile)
    
    // Check disk space
    if available, err := getDiskSpace(lm.config.Directory); err != nil {
        return fmt.Errorf("cannot check disk space: %w", err)
    } else if available < 100*1024*1024 { // 100MB minimum
        return fmt.Errorf("insufficient disk space: %d bytes available", available)
    }
    
    return nil
}
```

## Development Workflow

### Local Development

```bash
# Development configuration
LOG_LEVEL=debug
LOG_FORMAT=console
LOG_OUTPUT=console
GIN_MODE=debug

# Start with verbose logging
make up
make logs
```

### Testing

```bash
# Test with different log levels
LOG_LEVEL=error go test ./...

# Test log output
LOG_OUTPUT=file go test ./internal/logging/...

# Performance testing with logging disabled
LOG_LEVEL=fatal go test -bench=. ./...
```

### Debugging

```bash
# Enable detailed logging for specific components
LOG_LEVEL=debug LOG_CALLER_INFO=true go run cmd/server/main.go

# Filter logs for specific operations
docker-compose logs bankapi | jq 'select(.operation == "transfer")'

# Trace specific requests
docker-compose logs bankapi | jq 'select(.request_id == "20240115103045-abc12345")'
```

## Common Patterns

### Request Lifecycle Logging

```go
// Start of request
logger.Info().
    Str("request_id", requestID).
    Str("method", r.Method).
    Str("path", r.URL.Path).
    Str("client_ip", getClientIP(r)).
    Msg("Request started")

// End of request
logger.Info().
    Str("request_id", requestID).
    Int("status_code", statusCode).
    Dur("duration", time.Since(start)).
    Int64("response_size", responseSize).
    Msg("Request completed")
```

### Database Operation Logging

```go
// Before database operation
logger.Debug().
    Str("query", sanitizeQuery(query)).
    Interface("params", sanitizeParams(params)).
    Msg("Executing database query")

// After database operation
logger.Info().
    Str("operation", "select").
    Dur("duration", elapsed).
    Int64("rows_affected", rowsAffected).
    Msg("Database query completed")
```

### Background Job Logging

```go
// Job start
logger.Info().
    Str("job_id", jobID).
    Str("job_type", "send_email").
    Interface("payload", sanitizePayload(payload)).
    Msg("Background job started")

// Job completion
logger.Info().
    Str("job_id", jobID).
    Str("result", "success").
    Dur("duration", elapsed).
    Msg("Background job completed")
```

## Troubleshooting

### Common Issues

1. **High log volume**: Enable sampling or increase log level
2. **Missing context**: Ensure middleware is properly configured
3. **Performance impact**: Use conditional logging for expensive operations
4. **Disk space**: Configure appropriate retention and compression
5. **Missing logs**: Check file permissions and disk space

### Debug Commands

```bash
# Check logging configuration
env | grep LOG_

# Test log file creation
touch logs/test.log && rm logs/test.log

# Monitor log file growth
watch -n 1 'ls -lah logs/'

# Analyze log patterns
grep -E "(error|warn)" logs/app-*.log | wc -l
```

## Migration from Other Logging Libraries

### From Logrus

The migration from logrus to zerolog is complete. Key differences:

- **Performance**: Zerolog has zero allocations vs logrus allocations
- **API**: Chained API vs structured fields
- **Levels**: Slightly different level names
- **Output**: Native JSON support vs formatters

### Best Practices for Migration

1. **Update import statements**: Replace logrus imports with zerolog
2. **Update log calls**: Use zerolog's chained API
3. **Update configuration**: Use new environment variables
4. **Test thoroughly**: Ensure all log output is correct
5. **Monitor performance**: Verify performance improvements

## Conclusion

Following these best practices will help you:

- Maintain high application performance
- Ensure comprehensive monitoring and debugging capabilities
- Meet security and compliance requirements
- Provide excellent operational visibility

For more information, see:
- [Zerolog Documentation](https://github.com/rs/zerolog)
- [API Documentation](API.md)
- [Troubleshooting Guide](TROUBLESHOOTING.md)