# Design Document

## Overview

The Admin Dashboard System is a comprehensive administrative interface for the BankGo banking application. It consists of two main components: a React-based Single Page Application (SPA) frontend and a dedicated Go-based admin API backend. The system provides full administrative capabilities including user management, transaction monitoring, system health tracking, and database operations, all wrapped in a beautiful, responsive interface with an engaging loading animation.

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Admin Dashboard SPA                        │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────┐ │
│  │   React     │ │  TypeScript │ │   Tailwind  │ │  Framer   │ │
│  │ Components  │ │   Types     │ │     CSS     │ │  Motion   │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └───────────┘ │
└─────────────────────┬───────────────────────────────────────────┘
                      │ HTTPS/WebSocket
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Admin API Gateway                           │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────┐ │
│  │    Gin      │ │    Auth     │ │  WebSocket  │ │   CORS    │ │
│  │   Router    │ │ Middleware  │ │   Handler   │ │ Middleware│ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └───────────┘ │
└─────────────────────┬───────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Banking System APIs                           │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────┐ │
│  │   User      │ │  Account    │ │  Transfer   │ │ Database  │ │
│  │  Service    │ │  Service    │ │  Service    │ │  Direct   │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └───────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Container Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Docker Network                            │
│                                                                 │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────┐ │
│  │  Admin SPA      │    │   Admin API     │    │   Banking   │ │
│  │  Container      │    │   Container     │    │   System    │ │
│  │                 │    │                 │    │  Container  │ │
│  │  - React App    │◄──►│  - Go API       │◄──►│             │ │
│  │  - Nginx        │    │  - WebSocket    │    │  - Main API │ │
│  │  - Port 3000    │    │  - Port 8081    │    │  - Port 8080│ │
│  └─────────────────┘    └─────────────────┘    └─────────────┘ │
│                                                                 │
│  ┌─────────────────┐    ┌─────────────────┐                    │
│  │   PostgreSQL    │    │     Redis       │                    │
│  │   Database      │    │     Cache       │                    │
│  │   Port 5432     │    │   Port 6379     │                    │
│  └─────────────────┘    └─────────────────┘                    │
└─────────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### Frontend Components

#### 1. Loading Animation Component (`LoadingAnimation.tsx`)

**Purpose**: Displays the beautiful "BankGo" water-filling animation

**Key Features**:
- SVG-based text outline for crisp rendering
- CSS animations for water-filling effect
- Gradient colors representing money themes
- Responsive design for all screen sizes

**Technical Implementation**:
```typescript
interface LoadingAnimationProps {
  onComplete: () => void;
  duration?: number;
}

const LoadingAnimation: React.FC<LoadingAnimationProps> = ({
  onComplete,
  duration = 3000
}) => {
  // SVG text with clipping path for water effect
  // CSS keyframe animations for filling and shimmer
  // Gradient definitions for money-themed colors
}
```

**Animation Sequence**:
1. **Phase 1 (0-500ms)**: Display empty "BankGo" text outline
2. **Phase 2 (500-2500ms)**: Water fills from bottom to top with wave motion
3. **Phase 3 (2500-3000ms)**: Shimmer effect and transition preparation
4. **Phase 4 (3000ms+)**: Fade to dashboard

#### 2. Dashboard Layout Component (`DashboardLayout.tsx`)

**Purpose**: Main layout wrapper with navigation and content areas

**Key Features**:
- Responsive sidebar navigation
- Header with user info and notifications
- Main content area with routing
- Real-time notification system

#### 3. User Management Components

- `UserList.tsx`: Paginated user listing with search/filter
- `UserDetail.tsx`: Individual user information and actions
- `UserForm.tsx`: Create/edit user forms with validation

#### 4. System Monitoring Components

- `SystemHealth.tsx`: Real-time health metrics dashboard
- `MetricsChart.tsx`: Interactive charts for performance data
- `AlertPanel.tsx`: Real-time alerts and notifications

#### 5. Database Management Components

- `TableList.tsx`: Database table browser
- `TableView.tsx`: CRUD operations for table records
- `QueryBuilder.tsx`: Advanced search and filtering

### Backend API Structure

#### 1. Admin API Server (`cmd/admin/main.go`)

**Purpose**: Dedicated admin API server separate from main banking API

**Key Features**:
- Independent authentication system
- WebSocket support for real-time updates
- Proxy endpoints to main banking system
- Direct database access for admin operations

#### 2. Admin Handlers (`internal/admin/handlers/`)

**Endpoints**:
```go
// Authentication
POST   /api/admin/auth/login
POST   /api/admin/auth/logout
GET    /api/admin/auth/session

// User Management
GET    /api/admin/users
GET    /api/admin/users/:id
POST   /api/admin/users
PUT    /api/admin/users/:id
DELETE /api/admin/users/:id

// System Monitoring
GET    /api/admin/health
GET    /api/admin/metrics
GET    /api/admin/alerts
WebSocket /api/admin/ws/notifications

// Database Operations
GET    /api/admin/tables
GET    /api/admin/tables/:table/records
POST   /api/admin/tables/:table/records
PUT    /api/admin/tables/:table/records/:id
DELETE /api/admin/tables/:table/records/:id

// Transaction Management
GET    /api/admin/transactions
GET    /api/admin/transactions/:id
POST   /api/admin/transactions/:id/reverse
```

#### 3. Admin Services (`internal/admin/services/`)

**Services**:
- `AdminAuthService`: Session management and authentication
- `UserManagementService`: User CRUD operations
- `SystemMonitoringService`: Health metrics and alerts
- `DatabaseService`: Direct database operations
- `NotificationService`: Real-time WebSocket notifications

## Data Models

### Frontend State Models

```typescript
// Authentication State
interface AdminSession {
  isAuthenticated: boolean;
  pasetoToken: string;
  expiresAt: Date;
  adminInfo: AdminInfo;
}

interface AdminInfo {
  id: string;
  username: string;
  lastLogin: Date;
  permissions: string[];
}

// System Health State
interface SystemHealth {
  status: 'healthy' | 'warning' | 'critical';
  metrics: {
    cpuUsage: number;
    memoryUsage: number;
    dbConnections: number;
    apiResponseTime: number;
  };
  alerts: Alert[];
}

interface Alert {
  id: string;
  severity: 'critical' | 'warning' | 'info';
  message: string;
  timestamp: Date;
  acknowledged: boolean;
}

// Database Management State
interface TableSchema {
  name: string;
  columns: Column[];
  primaryKey: string[];
  foreignKeys: ForeignKey[];
}

interface Column {
  name: string;
  type: string;
  nullable: boolean;
  defaultValue?: any;
}
```

### Backend Models

```go
// Admin Authentication
type AdminSession struct {
    ID           string    `json:"id"`
    Username     string    `json:"username"`
    PasetoToken  string    `json:"paseto_token"`
    ExpiresAt    time.Time `json:"expires_at"`
    CreatedAt    time.Time `json:"created_at"`
}

// System Metrics
type SystemMetrics struct {
    CPUUsage        float64 `json:"cpu_usage"`
    MemoryUsage     float64 `json:"memory_usage"`
    DBConnections   int     `json:"db_connections"`
    APIResponseTime float64 `json:"api_response_time"`
    Timestamp       time.Time `json:"timestamp"`
}

// Database Operations
type TableRecord struct {
    TableName string                 `json:"table_name"`
    Data      map[string]interface{} `json:"data"`
    Metadata  RecordMetadata         `json:"metadata"`
}

type RecordMetadata struct {
    PrimaryKey   map[string]interface{} `json:"primary_key"`
    CreatedAt    *time.Time            `json:"created_at,omitempty"`
    UpdatedAt    *time.Time            `json:"updated_at,omitempty"`
}
```

## Error Handling

### Frontend Error Handling

**Error Boundary Component**:
```typescript
class AdminErrorBoundary extends React.Component {
  // Catches and displays user-friendly error messages
  // Provides retry mechanisms for recoverable errors
  // Logs errors for debugging
}
```

**API Error Handling**:
```typescript
interface APIError {
  code: string;
  message: string;
  details?: Record<string, any>;
}

// Centralized error handling with toast notifications
// Automatic retry for network errors
// Session refresh for authentication errors
```

### Backend Error Handling

**Structured Error Responses**:
```go
type AdminErrorResponse struct {
    Error     string            `json:"error"`
    Message   string            `json:"message"`
    Code      string            `json:"code"`
    Details   map[string]string `json:"details,omitempty"`
    Timestamp time.Time         `json:"timestamp"`
}
```

**Error Categories**:
- `admin_auth_failed`: Authentication/authorization errors
- `admin_validation_error`: Input validation failures
- `admin_permission_denied`: Insufficient permissions
- `admin_system_error`: Internal system errors
- `admin_database_error`: Database operation failures

## Testing Strategy

### Frontend Testing

**Unit Tests**:
- Component rendering and behavior
- State management logic
- Utility functions
- Animation timing and transitions

**Integration Tests**:
- API communication
- Authentication flows
- Real-time WebSocket connections
- Cross-component interactions

**E2E Tests**:
- Complete admin workflows
- Loading animation display
- Responsive design validation
- Performance benchmarks

### Backend Testing

**Unit Tests**:
- Handler logic and validation
- Service layer business logic
- Database operations
- Authentication mechanisms

**Integration Tests**:
- API endpoint functionality
- Database transactions
- WebSocket connections
- Main banking system integration

**Performance Tests**:
- Concurrent admin sessions
- Large dataset operations
- Real-time notification delivery
- Database query optimization

### Loading Animation Testing

**Visual Tests**:
- Animation smoothness across browsers
- Color accuracy and gradients
- Responsive behavior on different screens
- Performance impact measurement

**Functional Tests**:
- Animation completion triggers
- Transition to dashboard
- Loading state management
- Error handling during loading

## Security Considerations

### Authentication & Authorization

**Session Management**:
- PASETO v2 tokens for secure, stateless authentication (consistent with main banking system)
- 1-hour token expiration with automatic renewal
- Secure token storage in httpOnly cookies
- CSRF protection for state-changing operations
- Token invalidation on logout

**Permission System**:
- Role-based access control (RBAC)
- Granular permissions for different admin functions
- Audit logging for all administrative actions
- IP-based access restrictions (configurable)

### Data Protection

**Sensitive Data Handling**:
- Encryption of sensitive data in transit and at rest
- Masking of sensitive information in logs
- Secure deletion of temporary data
- Compliance with data protection regulations

**API Security**:
- Rate limiting for admin endpoints
- Input validation and sanitization
- SQL injection prevention
- XSS protection in frontend

## Performance Optimization

### Frontend Performance

**Loading Optimization**:
- Code splitting for route-based chunks
- Lazy loading of heavy components
- Image optimization and compression
- Service worker for caching

**Animation Performance**:
- Hardware-accelerated CSS animations
- Optimized SVG rendering
- Minimal DOM manipulation
- 60fps animation targets

**Data Management**:
- Virtual scrolling for large datasets
- Debounced search inputs
- Optimistic UI updates
- Efficient state management

### Backend Performance

**API Optimization**:
- Connection pooling for database operations
- Caching of frequently accessed data
- Pagination for large result sets
- Compression of API responses

**Real-time Features**:
- Efficient WebSocket connection management
- Message queuing for high-volume notifications
- Connection pooling and load balancing
- Graceful degradation for connection issues

## Deployment Strategy

### Container Configuration

**Admin SPA Container**:
```dockerfile
FROM node:18-alpine AS builder
# Build React application with production optimizations

FROM nginx:alpine
# Serve static files with optimized nginx configuration
# Include security headers and compression
```

**Admin API Container**:
```dockerfile
FROM golang:1.24-alpine AS builder
# Build Go binary with optimizations

FROM alpine:latest
# Minimal runtime with security updates
# Health check endpoints
```

### Docker Compose Integration

**Development Environment**:
```yaml
services:
  admin-spa:
    build: ./admin-frontend
    ports:
      - "3000:80"
    environment:
      - REACT_APP_API_URL=http://localhost:8081
    
  admin-api:
    build: ./admin-backend
    ports:
      - "8081:8081"
    environment:
      - BANKING_API_URL=http://banking-api:8080
      - DB_HOST=postgres
    depends_on:
      - postgres
      - redis
```

### Production Considerations

**Scaling**:
- Horizontal scaling for admin API instances
- Load balancing with session affinity
- CDN integration for static assets
- Database read replicas for reporting

**Monitoring**:
- Health check endpoints for containers
- Metrics collection and alerting
- Log aggregation and analysis
- Performance monitoring and profiling

## Integration Points

### Banking System Integration

**API Communication**:
- PASETO v2 token-based service-to-service authentication
- Shared PASETO secret key with main banking system for token validation
- Request/response logging and monitoring
- Circuit breaker pattern for resilience
- Timeout and retry configurations

**Data Synchronization**:
- Real-time updates via WebSocket
- Event-driven architecture for notifications
- Eventual consistency handling
- Conflict resolution strategies

### External Services

**Monitoring Integration**:
- Prometheus metrics collection
- Grafana dashboard integration
- Alert manager for critical events
- Log shipping to centralized systems

**Security Integration**:
- LDAP/Active Directory integration (future)
- Multi-factor authentication support
- Security scanning and vulnerability management
- Compliance reporting and auditing