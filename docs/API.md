# Bank REST API Documentation

## Overview

The Bank REST API provides comprehensive banking functionality including user authentication, multi-currency account management, and secure money transfers. All operations follow REST conventions with consistent JSON responses and proper HTTP status codes.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

The API uses PASETO (Platform-Agnostic Security Tokens) for authentication. After successful login, include the token in the Authorization header for all protected endpoints:

```
Authorization: Bearer <your-paseto-token>
```

## Content Type

All requests with body content must include:

```
Content-Type: application/json
```

## Response Format

### Success Response

```json
{
  "data": {
    // Response data
  },
  "message": "Success message (optional)"
}
```

### Error Response

```json
{
  "error": "error_type",
  "message": "Human readable error message",
  "code": 400,
  "details": {
    "field_name": "Specific validation error"
  }
}
```

## Endpoints

### Authentication

#### Register User

Creates a new user account and triggers welcome email processing.

**Endpoint:** `POST /auth/register`

**Request Body:**
```json
{
  "email": "john.doe@example.com",
  "password": "SecurePassword123!",
  "first_name": "John",
  "last_name": "Doe"
}
```

**Validation Rules:**
- `email`: Valid email format, unique
- `password`: Minimum 8 characters
- `first_name`: Required, max 100 characters
- `last_name`: Required, max 100 characters

**Success Response (201):**
```json
{
  "data": {
    "id": 1,
    "email": "john.doe@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "welcome_email_sent": false,
    "created_at": "2024-01-15T10:30:00Z"
  },
  "message": "User registered successfully"
}
```

**Error Responses:**
- `400`: Validation errors
- `409`: Email already exists

#### Login User

Authenticates user and returns PASETO token. Queues welcome email for first-time login.

**Endpoint:** `POST /auth/login`

**Request Body:**
```json
{
  "email": "john.doe@example.com",
  "password": "SecurePassword123!"
}
```

**Success Response (200):**
```json
{
  "data": {
    "token": "v2.local.xxx...",
    "user": {
      "id": 1,
      "email": "john.doe@example.com",
      "first_name": "John",
      "last_name": "Doe",
      "welcome_email_sent": true
    }
  },
  "message": "Login successful"
}
```

**Error Responses:**
- `400`: Validation errors
- `401`: Invalid credentials

#### Logout User

Invalidates the current session token.

**Endpoint:** `POST /auth/logout`

**Headers:** `Authorization: Bearer <token>`

**Success Response (200):**
```json
{
  "message": "Logout successful"
}
```

### Account Management

#### Create Account

Creates a new account for the authenticated user with specified currency.

**Endpoint:** `POST /accounts`

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "currency": "USD"
}
```

**Validation Rules:**
- `currency`: Exactly 3 characters, valid currency code
- One account per currency per user

**Success Response (201):**
```json
{
  "data": {
    "id": 1,
    "user_id": 1,
    "currency": "USD",
    "balance": "0.00",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  },
  "message": "Account created successfully"
}
```

**Error Responses:**
- `400`: Invalid currency format
- `409`: Account with this currency already exists
- `401`: Unauthorized

#### List User Accounts

Returns all accounts belonging to the authenticated user.

**Endpoint:** `GET /accounts`

**Headers:** `Authorization: Bearer <token>`

**Success Response (200):**
```json
{
  "data": [
    {
      "id": 1,
      "user_id": 1,
      "currency": "USD",
      "balance": "1500.50",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T14:20:00Z"
    },
    {
      "id": 2,
      "user_id": 1,
      "currency": "EUR",
      "balance": "750.25",
      "created_at": "2024-01-15T11:00:00Z",
      "updated_at": "2024-01-15T13:45:00Z"
    }
  ]
}
```

#### Get Account Details

Returns details for a specific account. Users can only access their own accounts.

**Endpoint:** `GET /accounts/{id}`

**Headers:** `Authorization: Bearer <token>`

**Path Parameters:**
- `id`: Account ID (integer)

**Success Response (200):**
```json
{
  "data": {
    "id": 1,
    "user_id": 1,
    "currency": "USD",
    "balance": "1500.50",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T14:20:00Z"
  }
}
```

**Error Responses:**
- `404`: Account not found
- `403`: Account belongs to different user
- `401`: Unauthorized

#### Update Account

Updates account details (administrative operation, balance cannot be modified directly).

**Endpoint:** `PUT /accounts/{id}`

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "currency": "USD"
}
```

**Success Response (200):**
```json
{
  "data": {
    "id": 1,
    "user_id": 1,
    "currency": "USD",
    "balance": "1500.50",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T15:30:00Z"
  },
  "message": "Account updated successfully"
}
```

#### Delete Account

Deletes an account. Only accounts with zero balance and no transaction history can be deleted.

**Endpoint:** `DELETE /accounts/{id}`

**Headers:** `Authorization: Bearer <token>`

**Success Response (204):** No content

**Error Responses:**
- `422`: Account has non-zero balance or transaction history
- `404`: Account not found
- `403`: Account belongs to different user

### Money Transfers

#### Create Transfer

Transfers money between accounts with atomic database transaction.

**Endpoint:** `POST /transfers`

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "from_account_id": 1,
  "to_account_id": 2,
  "amount": "100.50",
  "description": "Payment for services"
}
```

**Validation Rules:**
- `from_account_id`: Must exist and belong to user
- `to_account_id`: Must exist
- `amount`: Positive decimal, max 2 decimal places
- `description`: Optional, max 255 characters
- Both accounts must have same currency
- Source account must have sufficient balance

**Success Response (201):**
```json
{
  "data": {
    "id": 1,
    "from_account_id": 1,
    "to_account_id": 2,
    "amount": "100.50",
    "description": "Payment for services",
    "status": "completed",
    "created_at": "2024-01-15T15:30:00Z"
  },
  "message": "Transfer completed successfully"
}
```

**Error Responses:**
- `422`: Insufficient balance, currency mismatch
- `404`: Account not found
- `403`: Unauthorized access to account
- `400`: Validation errors

#### Get Transfer History

Returns transfer history for accounts belonging to the authenticated user.

**Endpoint:** `GET /transfers`

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**
- `account_id` (optional): Filter by specific account
- `limit` (optional): Number of results (default: 50, max: 100)
- `offset` (optional): Pagination offset (default: 0)

**Success Response (200):**
```json
{
  "data": [
    {
      "id": 1,
      "from_account_id": 1,
      "to_account_id": 2,
      "amount": "100.50",
      "description": "Payment for services",
      "status": "completed",
      "created_at": "2024-01-15T15:30:00Z"
    },
    {
      "id": 2,
      "from_account_id": 3,
      "to_account_id": 1,
      "amount": "250.00",
      "description": "Refund",
      "status": "completed",
      "created_at": "2024-01-15T14:20:00Z"
    }
  ],
  "pagination": {
    "limit": 50,
    "offset": 0,
    "total": 2
  }
}
```

#### Get Transfer Details

Returns details for a specific transfer.

**Endpoint:** `GET /transfers/{id}`

**Headers:** `Authorization: Bearer <token>`

**Success Response (200):**
```json
{
  "data": {
    "id": 1,
    "from_account_id": 1,
    "to_account_id": 2,
    "amount": "100.50",
    "description": "Payment for services",
    "status": "completed",
    "created_at": "2024-01-15T15:30:00Z"
  }
}
```

### Health Check

#### Service Health

Returns the health status of the service and its dependencies.

**Endpoint:** `GET /health`

**Success Response (200):**
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

**Unhealthy Response (503):**
```json
{
  "status": "unhealthy",
  "version": "v1.0.0",
  "timestamp": "2024-01-15T15:30:00Z",
  "services": {
    "database": {
      "status": "disconnected",
      "error": "connection timeout"
    },
    "redis": {
      "status": "connected",
      "response_time": "1ms"
    }
  }
}
```

## HTTP Status Codes

| Code | Description | Usage |
|------|-------------|-------|
| 200 | OK | Successful GET, PUT requests |
| 201 | Created | Successful POST requests |
| 204 | No Content | Successful DELETE requests |
| 400 | Bad Request | Validation errors, malformed requests |
| 401 | Unauthorized | Missing or invalid authentication |
| 403 | Forbidden | Valid auth but insufficient permissions |
| 404 | Not Found | Resource doesn't exist |
| 409 | Conflict | Resource already exists |
| 422 | Unprocessable Entity | Business logic errors |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server errors |
| 503 | Service Unavailable | Service health check failed |

## Rate Limiting

The API implements rate limiting to prevent abuse:

- **Per IP**: 100 requests per minute
- **Per User**: 1000 requests per hour (authenticated endpoints)

When rate limit is exceeded, the API returns:

```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests",
  "code": 429,
  "details": {
    "retry_after": "60s"
  }
}
```

## Business Rules

### Account Management
1. Users can create multiple accounts with different currencies
2. Only one account per currency per user is allowed
3. Account deletion requires zero balance and no transaction history
4. Users can only access their own accounts

### Money Transfers
1. Both accounts must have the same currency
2. Source account must have sufficient balance
3. All transfer operations are atomic (database transactions)
4. Failed transfers are automatically rolled back
5. Transfer history is maintained for all accounts

### Authentication
1. PASETO tokens expire after 24 hours (configurable)
2. Welcome emails are sent on first login
3. Email processing is handled asynchronously
4. Failed email deliveries are retried automatically

## Examples

### Complete User Journey

1. **Register a new user:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "SecurePass123!",
    "first_name": "Alice",
    "last_name": "Johnson"
  }'
```

2. **Login to get token:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "SecurePass123!"
  }'
```

3. **Create USD account:**
```bash
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"currency": "USD"}'
```

4. **Create EUR account:**
```bash
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"currency": "EUR"}'
```

5. **List all accounts:**
```bash
curl -X GET http://localhost:8080/api/v1/accounts \
  -H "Authorization: Bearer <token>"
```

6. **Transfer money (after funding accounts):**
```bash
curl -X POST http://localhost:8080/api/v1/transfers \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "from_account_id": 1,
    "to_account_id": 2,
    "amount": "100.00",
    "description": "Internal transfer"
  }'
```

### Error Handling Examples

**Insufficient Balance:**
```bash
# Response (422)
{
  "error": "insufficient_balance",
  "message": "Account does not have sufficient balance for this transfer",
  "code": 422,
  "details": {
    "available_balance": "50.00",
    "requested_amount": "100.00"
  }
}
```

**Currency Mismatch:**
```bash
# Response (422)
{
  "error": "currency_mismatch",
  "message": "Source and destination accounts must have the same currency",
  "code": 422,
  "details": {
    "source_currency": "USD",
    "destination_currency": "EUR"
  }
}
```

**Validation Error:**
```bash
# Response (400)
{
  "error": "validation_failed",
  "message": "Request validation failed",
  "code": 400,
  "details": {
    "currency": "Currency must be exactly 3 characters",
    "amount": "Amount must be greater than 0"
  }
}
```