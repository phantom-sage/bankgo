import { ApiResponse, PaginatedResponse } from '../types';
import { authService } from './authService';

// Base API configuration
const API_BASE_URL =
  process.env.REACT_APP_API_URL || 'http://localhost:8081/api/admin';

class ApiService {
  private baseURL: string;

  constructor(baseURL: string = API_BASE_URL) {
    this.baseURL = baseURL;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    const url = `${this.baseURL}${endpoint}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string>),
    };

    // Get token from auth service
    const token = authService.getStoredToken();
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }

    try {
      const response = await fetch(url, {
        ...options,
        headers,
      });

      const data = await response.json();

      if (!response.ok) {
        // Handle authentication errors
        if (response.status === 401) {
          // Token might be expired, try to refresh session
          const sessionResult = await authService.validateSession();
          if (!sessionResult.success) {
            // Session is invalid, redirect to login will be handled by useAuth hook
            return {
              success: false,
              error: 'Authentication failed',
              message: 'Session expired. Please log in again.',
            };
          }
          
          // Retry the request with refreshed token
          const newToken = authService.getStoredToken();
          if (newToken) {
            headers.Authorization = `Bearer ${newToken}`;
            const retryResponse = await fetch(url, {
              ...options,
              headers,
            });
            
            if (retryResponse.ok) {
              const retryData = await retryResponse.json();
              return {
                success: true,
                data: retryData,
              };
            }
          }
        }

        return {
          success: false,
          error: data.error || 'An error occurred',
          message: data.message,
        };
      }

      return {
        success: true,
        data,
      };
    } catch (error) {
      return {
        success: false,
        error: 'Network error',
        message: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  // Authentication endpoints - will be implemented in task 12
  async login(username: string, password: string) {
    return this.request('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
  }

  async logout() {
    return this.request('/auth/logout', {
      method: 'POST',
    });
  }

  async getSession() {
    return this.request('/auth/session');
  }

  // User management endpoints - will be implemented in task 14
  async getUsers(page = 1, limit = 10, search?: string) {
    const params = new URLSearchParams({
      page: page.toString(),
      limit: limit.toString(),
    });

    if (search) {
      params.append('search', search);
    }

    return this.request<PaginatedResponse<any>>(`/users?${params}`);
  }

  // System monitoring endpoints - will be implemented in task 15
  async getSystemHealth() {
    return this.request('/health');
  }

  async getMetrics() {
    return this.request('/metrics');
  }

  // Database management endpoints - will be implemented in task 16
  async getTables() {
    return this.request('/tables');
  }

  // Transaction management endpoints - will be implemented in task 17
  async getTransactions(page = 1, limit = 10) {
    const params = new URLSearchParams({
      page: page.toString(),
      limit: limit.toString(),
    });

    return this.request<PaginatedResponse<any>>(`/transactions?${params}`);
  }
}

export const apiService = new ApiService();
export default ApiService;
