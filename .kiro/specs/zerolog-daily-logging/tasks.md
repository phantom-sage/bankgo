# Implementation Plan

- [x] 1. Set up zerolog dependencies and core configuration
  - Add zerolog and lumberjack dependencies to go.mod
  - Create logging configuration struct with environment variable support
  - Implement configuration loading and validation functions
  - _Requirements: 1, 4_

- [x] 2. Implement daily file writer with rotation
  - [x] 2.1 Create DailyFileWriter struct and methods
    - Write DailyFileWriter struct with rotation logic
    - Implement Write method with date-based rotation checking
    - Add file creation with timestamped naming (app-YYYY-MM-DD.log format)
    - Write unit tests for file writer and rotation logic
    - _Requirements: 2_

  - [x] 2.2 Add log file cleanup and retention policies
    - Implement cleanup methods for old log files based on MaxAge and MaxBackups
    - Add compression support for rotated log files
    - Create cleanup scheduling and automatic execution
    - Write unit tests for cleanup and retention functionality
    - _Requirements: 7_

- [x] 3. Create core logger manager and configuration
  - [x] 3.1 Implement LoggerManager with multi-output support
    - Create LoggerManager struct with zerolog logger initialization
    - Add support for console, file, and combined output modes
    - Implement different log formats (JSON for production, console for development)
    - Write unit tests for logger manager functionality
    - _Requirements: 1, 4_

  - [x] 3.2 Add contextual logging capabilities
    - Create ContextLogger struct for request-scoped logging
    - Implement methods for adding request ID, user context, and custom fields
    - Add correlation ID support for request tracing
    - Write unit tests for contextual logging features
    - _Requirements: 3_

- [x] 4. Implement specialized loggers for audit and performance
  - [x] 4.1 Create AuditLogger for security and compliance logging
    - Implement AuditLogger struct with specialized audit methods
    - Add methods for authentication, account operations, and transfer logging
    - Include administrative action logging and security event tracking
    - Write unit tests for audit logging functionality
    - _Requirements: 5_

  - [x] 4.2 Create PerformanceLogger for metrics and monitoring
    - Implement PerformanceLogger struct with performance tracking methods
    - Add HTTP request timing, database query performance, and background job metrics
    - Include resource usage monitoring and external service call tracking
    - Write unit tests for performance logging features
    - _Requirements: 6_

- [x] 5. Migrate HTTP middleware from logrus to zerolog
  - [x] 5.1 Update RequestLogger middleware to use zerolog
    - Replace logrus logger with zerolog in middleware configuration
    - Maintain existing sensitive data filtering and request body sanitization
    - Add request ID correlation and contextual field support
    - Update log format to use zerolog's structured logging
    - _Requirements: 1, 3_

  - [x] 5.2 Add performance metrics to request logging
    - Include response time percentiles and request size metrics
    - Add database query count and execution time tracking per request
    - Implement error rate monitoring and status code distribution logging
    - Write integration tests for updated middleware functionality
    - _Requirements: 6_

- [x] 6. Update service layer to use zerolog
  - [x] 6.1 Migrate UserService to use contextual zerolog
    - Update UserService constructor to accept zerolog logger
    - Add contextual logging to CreateUser, AuthenticateUser, and other methods
    - Include audit logging for user registration and authentication events
    - Replace existing log statements with structured zerolog calls
    - _Requirements: 3, 5_

  - [x] 6.2 Migrate AccountService with audit logging
    - Update AccountService constructor and methods to use zerolog
    - Add audit logging for account creation, updates, and deletion operations
    - Include performance logging for database operations
    - Add contextual logging with user and account information
    - _Requirements: 3, 5, 6_

  - [x] 6.3 Migrate TransferService with transaction logging
    - Update TransferService to use zerolog for transfer operations
    - Add detailed audit logging for money transfer transactions
    - Include performance metrics for database transaction execution
    - Add error logging with transaction rollback context
    - _Requirements: 3, 5, 6_

- [x] 7. Update database and repository layers
  - [x] 7.1 Add database operation logging to repositories
    - Update repository constructors to accept zerolog logger
    - Add query execution time logging and affected rows tracking
    - Include error logging with query context and parameters (sanitized)
    - Add connection pool monitoring and health check logging
    - _Requirements: 3, 6_

  - [x] 7.2 Implement database transaction logging
    - Add transaction begin, commit, and rollback logging
    - Include transaction duration and operation count tracking
    - Add deadlock detection and retry logging
    - Write integration tests for database logging functionality
    - _Requirements: 3, 6_

- [x] 8. Update background job processing with zerolog
  - [x] 8.1 Migrate Asyncq worker logging to zerolog
    - Update email worker to use zerolog for job processing
    - Add job execution timing and success/failure rate tracking
    - Include job queue health monitoring and backlog size logging
    - Add retry attempt logging with failure reason context
    - _Requirements: 3, 6_

  - [x] 8.2 Add background job performance monitoring
    - Implement job correlation IDs for end-to-end tracing
    - Add job queue depth monitoring and processing rate metrics
    - Include worker health status and resource usage logging
    - Write integration tests for background job logging
    - _Requirements: 6_

- [ ] 9. Update error handling throughout application
  - [ ] 9.1 Implement structured error logging
    - Create ErrorContext struct for consistent error logging
    - Update all error handling to use structured zerolog error logging
    - Add stack trace logging for critical errors
    - Include request context and user information in error logs
    - _Requirements: 3, 5_

  - [ ] 9.2 Add error categorization and monitoring
    - Implement error classification (validation, business logic, system errors)
    - Add error frequency tracking and alerting thresholds
    - Include error correlation with request IDs and user context
    - Write unit tests for error logging functionality
    - _Requirements: 5_

- [ ] 10. Update configuration and environment setup
  - [ ] 10.1 Add logging configuration to application config
    - Update main configuration struct to include LogConfig
    - Add environment variable loading for all logging settings
    - Include configuration validation and default value handling
    - Update .env.example with logging configuration variables
    - _Requirements: 4_

  - [ ] 10.2 Update application initialization with zerolog
    - Modify main.go to initialize LoggerManager with configuration
    - Update service constructors to receive logger instances
    - Add graceful shutdown handling for log file flushing
    - Include health check integration for logging system
    - _Requirements: 1, 4_

- [ ] 11. Add comprehensive testing for logging system
  - [ ] 11.1 Create unit tests for all logging components
    - Write tests for DailyFileWriter rotation and cleanup logic
    - Add tests for LoggerManager configuration and output modes
    - Include tests for contextual logging and field addition
    - Test sensitive data filtering and audit logging functionality
    - _Requirements: 1, 2, 3, 5, 7_

  - [ ] 11.2 Create integration tests for end-to-end logging
    - Write tests for complete request logging flow from HTTP to file
    - Add tests for log file creation, rotation, and cleanup
    - Include tests for concurrent logging and thread safety
    - Test logging performance under load and memory usage
    - _Requirements: 1, 2, 3, 6_

- [ ] 12. Remove logrus dependencies and clean up
  - [ ] 12.1 Remove logrus imports and dependencies
    - Remove logrus import statements from all files
    - Update go.mod to remove logrus dependency
    - Clean up any remaining logrus-specific code or configurations
    - Verify no logrus references remain in codebase
    - _Requirements: 1_

  - [ ] 12.2 Update documentation and examples
    - Update README.md with new logging configuration instructions
    - Add logging best practices and usage examples
    - Include troubleshooting guide for logging issues
    - Update API documentation with new log format examples
    - _Requirements: 4_