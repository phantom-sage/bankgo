// Authentication Types
export interface AdminSession {
  isAuthenticated: boolean;
  pasetoToken: string;
  expiresAt: Date;
  adminInfo: AdminInfo;
}

export interface AdminInfo {
  id: string;
  username: string;
  lastLogin: Date;
  permissions: string[];
}

export interface LoginCredentials {
  username: string;
  password: string;
}

// System Health Types
export interface SystemHealth {
  status: 'healthy' | 'warning' | 'critical';
  metrics: SystemMetrics;
  alerts: Alert[];
}

export interface SystemMetrics {
  cpuUsage: number;
  memoryUsage: number;
  dbConnections: number;
  apiResponseTime: number;
  timestamp: Date;
}

export interface Alert {
  id: string;
  severity: 'critical' | 'warning' | 'info';
  message: string;
  timestamp: Date;
  acknowledged: boolean;
  resolved?: boolean;
  resolvedAt?: Date;
  resolvedBy?: string;
}

// User Management Types
export interface User {
  id: string;
  username: string;
  email: string;
  firstName: string;
  lastName: string;
  isActive: boolean;
  createdAt: Date;
  updatedAt: Date;
  lastLogin?: Date;
}

export interface CreateUserRequest {
  username: string;
  email: string;
  firstName: string;
  lastName: string;
  password: string;
}

export interface UpdateUserRequest {
  email?: string;
  firstName?: string;
  lastName?: string;
  isActive?: boolean;
}

// Database Management Types
export interface TableSchema {
  name: string;
  columns: Column[];
  primaryKey: string[];
  foreignKeys: ForeignKey[];
  recordCount: number;
}

export interface Column {
  name: string;
  type: string;
  nullable: boolean;
  defaultValue?: any;
  isPrimaryKey: boolean;
  isForeignKey: boolean;
}

export interface ForeignKey {
  columnName: string;
  referencedTable: string;
  referencedColumn: string;
}

export interface TableRecord {
  tableName: string;
  data: Record<string, any>;
  metadata: RecordMetadata;
}

export interface RecordMetadata {
  primaryKey: Record<string, any>;
  createdAt?: Date;
  updatedAt?: Date;
}

// Transaction Types
export interface Transaction {
  id: string;
  fromAccountId: string;
  toAccountId: string;
  amount: number;
  currency: string;
  description?: string;
  status: 'pending' | 'completed' | 'failed' | 'reversed';
  createdAt: Date;
  completedAt?: Date;
  reversedAt?: Date;
  reversedBy?: string;
}

export interface Account {
  id: string;
  userId: string;
  currency: string;
  balance: number;
  isActive: boolean;
  createdAt: Date;
  updatedAt: Date;
}

// API Response Types
export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
  };
}

// Navigation Types
export interface NavItem {
  id: string;
  label: string;
  path: string;
  icon: string;
  children?: NavItem[];
}

// Form Types
export interface FormField {
  name: string;
  label: string;
  type:
    | 'text'
    | 'email'
    | 'password'
    | 'number'
    | 'select'
    | 'checkbox'
    | 'textarea';
  required?: boolean;
  placeholder?: string;
  options?: { value: string; label: string }[];
  validation?: {
    min?: number;
    max?: number;
    pattern?: string;
    message?: string;
  };
}

// WebSocket Types
export interface WebSocketMessage {
  type: 'alert' | 'notification' | 'system_update' | 'user_activity';
  payload: any;
  timestamp: Date;
}

// Loading Animation Types
export interface LoadingAnimationProps {
  onComplete: () => void;
  duration?: number;
}
