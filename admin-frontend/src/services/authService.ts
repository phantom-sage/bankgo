import { AdminSession, AdminInfo, LoginCredentials, ApiResponse } from '../types';

interface LoginResponse {
  token: string;
  expiresAt: string;
  adminInfo: AdminInfo;
}

interface SessionResponse {
  valid: boolean;
  adminInfo?: AdminInfo;
  expiresAt?: string;
}

class AuthService {
  private readonly API_BASE_URL: string;
  private readonly TOKEN_KEY = 'admin_paseto_token';
  private readonly SESSION_KEY = 'admin_session';
  private refreshTimer: NodeJS.Timeout | null = null;

  constructor() {
    this.API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8081/api/admin';
  }

  /**
   * Authenticate admin user with username/password
   */
  async login(credentials: LoginCredentials): Promise<ApiResponse<AdminSession>> {
    try {
      const response = await fetch(`${this.API_BASE_URL}/auth/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(credentials),
      });

      const data = await response.json();

      if (!response.ok) {
        return {
          success: false,
          error: data.error || 'Login failed',
          message: data.message || 'Invalid credentials',
        };
      }

      const loginData: LoginResponse = data;
      const session: AdminSession = {
        isAuthenticated: true,
        pasetoToken: loginData.token,
        expiresAt: new Date(loginData.expiresAt),
        adminInfo: loginData.adminInfo,
      };

      // Store session data
      this.storeSession(session);
      this.scheduleTokenRefresh(session.expiresAt);

      return {
        success: true,
        data: session,
      };
    } catch (error) {
      return {
        success: false,
        error: 'Network error',
        message: error instanceof Error ? error.message : 'Connection failed',
      };
    }
  }

  /**
   * Logout admin user and clear session
   */
  async logout(): Promise<void> {
    const token = this.getStoredToken();
    
    if (token) {
      try {
        await fetch(`${this.API_BASE_URL}/auth/logout`, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        });
      } catch (error) {
        console.warn('Logout request failed:', error);
      }
    }

    this.clearSession();
  }

  /**
   * Validate current session with backend
   */
  async validateSession(): Promise<ApiResponse<AdminSession>> {
    const token = this.getStoredToken();
    
    if (!token) {
      return {
        success: false,
        error: 'No token found',
        message: 'User not authenticated',
      };
    }

    try {
      const response = await fetch(`${this.API_BASE_URL}/auth/session`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });

      const data: SessionResponse = await response.json();

      if (!response.ok || !data.valid) {
        this.clearSession();
        return {
          success: false,
          error: 'Session invalid',
          message: 'Session has expired or is invalid',
        };
      }

      if (!data.adminInfo || !data.expiresAt) {
        this.clearSession();
        return {
          success: false,
          error: 'Invalid session data',
          message: 'Session data is incomplete',
        };
      }

      const session: AdminSession = {
        isAuthenticated: true,
        pasetoToken: token,
        expiresAt: new Date(data.expiresAt),
        adminInfo: data.adminInfo,
      };

      // Update stored session and refresh timer
      this.storeSession(session);
      this.scheduleTokenRefresh(session.expiresAt);

      return {
        success: true,
        data: session,
      };
    } catch (error) {
      this.clearSession();
      return {
        success: false,
        error: 'Network error',
        message: error instanceof Error ? error.message : 'Connection failed',
      };
    }
  }

  /**
   * Get stored session from localStorage
   */
  getStoredSession(): AdminSession | null {
    try {
      const sessionData = localStorage.getItem(this.SESSION_KEY);
      if (!sessionData) return null;

      const session: AdminSession = JSON.parse(sessionData);
      session.expiresAt = new Date(session.expiresAt);
      session.adminInfo.lastLogin = new Date(session.adminInfo.lastLogin);

      // Check if session is expired
      if (session.expiresAt <= new Date()) {
        this.clearSession();
        return null;
      }

      return session;
    } catch (error) {
      console.error('Error parsing stored session:', error);
      this.clearSession();
      return null;
    }
  }

  /**
   * Get stored PASETO token
   */
  getStoredToken(): string | null {
    return localStorage.getItem(this.TOKEN_KEY);
  }

  /**
   * Store session data in localStorage
   */
  private storeSession(session: AdminSession): void {
    localStorage.setItem(this.TOKEN_KEY, session.pasetoToken);
    localStorage.setItem(this.SESSION_KEY, JSON.stringify(session));
  }

  /**
   * Clear all session data
   */
  private clearSession(): void {
    localStorage.removeItem(this.TOKEN_KEY);
    localStorage.removeItem(this.SESSION_KEY);
    
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer);
      this.refreshTimer = null;
    }
  }

  /**
   * Schedule automatic token refresh before expiration
   */
  private scheduleTokenRefresh(expiresAt: Date): void {
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer);
    }

    const now = new Date().getTime();
    const expiry = expiresAt.getTime();
    const refreshTime = expiry - now - (5 * 60 * 1000); // Refresh 5 minutes before expiry

    if (refreshTime > 0) {
      this.refreshTimer = setTimeout(() => {
        this.validateSession().catch(console.error);
      }, refreshTime);
    }
  }

  /**
   * Check if session is close to expiring (within 10 minutes)
   */
  isSessionExpiringSoon(): boolean {
    const session = this.getStoredSession();
    if (!session) return false;

    const now = new Date().getTime();
    const expiry = session.expiresAt.getTime();
    const tenMinutes = 10 * 60 * 1000;

    return (expiry - now) <= tenMinutes;
  }

  /**
   * Get time remaining until session expires
   */
  getTimeUntilExpiry(): number {
    const session = this.getStoredSession();
    if (!session) return 0;

    const now = new Date().getTime();
    const expiry = session.expiresAt.getTime();
    
    return Math.max(0, expiry - now);
  }
}

export const authService = new AuthService();
export default AuthService;