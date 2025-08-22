# Product Overview

## Bank REST API

A production-ready banking REST API service that provides comprehensive financial functionality including multi-currency account management, secure money transfers, user authentication, and background email processing.

## Core Features

- **Multi-currency account management**: Users can create and manage accounts in different currencies with unique constraints (one account per currency per user)
- **Secure money transfers**: Atomic database transactions with automatic rollback capability between accounts of the same currency
- **PASETO authentication**: Secure, stateless token-based authentication system with configurable expiration
- **Background email processing**: Asynchronous welcome emails using Redis and Asyncq with retry logic
- **Production-ready security**: Rate limiting, CORS, request logging, and comprehensive security headers

## Business Rules

### Account Management
- Users can create multiple accounts with different currencies
- Only one account per currency per user is allowed
- Account deletion requires zero balance and no transaction history
- Users can only access their own accounts

### Money Transfers
- Both accounts must have the same currency
- Source account must have sufficient balance
- All transfer operations are atomic (database transactions)
- Failed transfers are automatically rolled back
- Transfer history is maintained for all accounts

### Authentication & Email
- PASETO tokens expire after 24 hours (configurable)
- Welcome emails are sent on first login
- Email processing is handled asynchronously with retry logic
- Failed email deliveries are retried automatically

## Development Methodology

This project follows **Spec-Driven Development** methodology where all features are developed through a structured process of requirements gathering, design documentation, implementation planning, and iterative development with testing.