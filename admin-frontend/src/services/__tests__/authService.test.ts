import { authService } from '../authService';
import { AdminSession, LoginCredentials } from '../../types';

// Mock fetch globally
global.fetch = jest.fn();

// Mock localStorage
const localStorageMock = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  removeItem: jest.fn(),
  clear: jest.fn(),
};
Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
});

describe('AuthService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    localStorageMock.getItem.mockClear();
    localStorageMock.setItem.mockClear();
    localStorageMock.removeItem.mockClear();
  });

  describe('login', () => {
    const mockCredentials: LoginCredentials = {
      username: 'admin',
      password: 'admin',
    };

    const mockLoginResponse = {
      token: 'mock-paseto-token',
      expiresAt: new Date(Date.now() + 3600000).toISOString(), // 1 hour from now
      adminInfo: {
        id: '1',
        username: 'admin',
        lastLogin: new Date().toISOString(),
        permissions: ['admin'],
      },
    };

    it('should successfully login with valid credentials', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => mockLoginResponse,
      });

      const result = await authService.login(mockCredentials);

      expect(result.success).toBe(true);
      expect(result.data).toBeDefined();
      expect(result.data?.isAuthenticated).toBe(true);
      expect(result.data?.pasetoToken).toBe(mockLoginResponse.token);
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        'admin_paseto_token',
        mockLoginResponse.token
      );
    });

    it('should handle login failure with invalid credentials', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        json: async () => ({
          error: 'authentication_failed',
          message: 'Invalid credentials',
        }),
      });

      const result = await authService.login(mockCredentials);

      expect(result.success).toBe(false);
      expect(result.error).toBe('authentication_failed');
      expect(result.message).toBe('Invalid credentials');
      expect(localStorageMock.setItem).not.toHaveBeenCalled();
    });

    it('should handle network errors during login', async () => {
      (fetch as jest.Mock).mockRejectedValueOnce(new Error('Network error'));

      const result = await authService.login(mockCredentials);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Network error');
      expect(result.message).toBe('Network error');
    });
  });

  describe('logout', () => {
    it('should successfully logout and clear session data', async () => {
      localStorageMock.getItem.mockReturnValue('mock-token');
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
      });

      await authService.logout();

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/auth/logout'),
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            Authorization: 'Bearer mock-token',
          }),
        })
      );
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('admin_paseto_token');
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('admin_session');
    });

    it('should clear session even if logout request fails', async () => {
      localStorageMock.getItem.mockReturnValue('mock-token');
      (fetch as jest.Mock).mockRejectedValueOnce(new Error('Network error'));

      await authService.logout();

      expect(localStorageMock.removeItem).toHaveBeenCalledWith('admin_paseto_token');
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('admin_session');
    });
  });

  describe('validateSession', () => {
    it('should validate a valid session', async () => {
      const mockToken = 'valid-token';
      localStorageMock.getItem.mockReturnValue(mockToken);

      const mockSessionResponse = {
        valid: true,
        adminInfo: {
          id: '1',
          username: 'admin',
          lastLogin: new Date().toISOString(),
          permissions: ['admin'],
        },
        expiresAt: new Date(Date.now() + 3600000).toISOString(),
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => mockSessionResponse,
      });

      const result = await authService.validateSession();

      expect(result.success).toBe(true);
      expect(result.data?.isAuthenticated).toBe(true);
      expect(result.data?.pasetoToken).toBe(mockToken);
    });

    it('should handle invalid session', async () => {
      localStorageMock.getItem.mockReturnValue('invalid-token');

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        json: async () => ({
          valid: false,
          error: 'Invalid session',
        }),
      });

      const result = await authService.validateSession();

      expect(result.success).toBe(false);
      expect(result.error).toBe('Session invalid');
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('admin_paseto_token');
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('admin_session');
    });

    it('should return error when no token is stored', async () => {
      localStorageMock.getItem.mockReturnValue(null);

      const result = await authService.validateSession();

      expect(result.success).toBe(false);
      expect(result.error).toBe('No token found');
      expect(fetch).not.toHaveBeenCalled();
    });
  });

  describe('getStoredSession', () => {
    it('should return stored session when valid', () => {
      const mockSession: AdminSession = {
        isAuthenticated: true,
        pasetoToken: 'mock-token',
        expiresAt: new Date(Date.now() + 3600000), // 1 hour from now
        adminInfo: {
          id: '1',
          username: 'admin',
          lastLogin: new Date(),
          permissions: ['admin'],
        },
      };

      localStorageMock.getItem.mockReturnValue(JSON.stringify(mockSession));

      const result = authService.getStoredSession();

      expect(result).toBeDefined();
      expect(result?.isAuthenticated).toBe(true);
      expect(result?.pasetoToken).toBe('mock-token');
    });

    it('should return null when session is expired', () => {
      const expiredSession: AdminSession = {
        isAuthenticated: true,
        pasetoToken: 'mock-token',
        expiresAt: new Date(Date.now() - 1000), // 1 second ago
        adminInfo: {
          id: '1',
          username: 'admin',
          lastLogin: new Date(),
          permissions: ['admin'],
        },
      };

      localStorageMock.getItem.mockReturnValue(JSON.stringify(expiredSession));

      const result = authService.getStoredSession();

      expect(result).toBeNull();
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('admin_paseto_token');
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('admin_session');
    });

    it('should return null when no session is stored', () => {
      localStorageMock.getItem.mockReturnValue(null);

      const result = authService.getStoredSession();

      expect(result).toBeNull();
    });

    it('should handle corrupted session data', () => {
      localStorageMock.getItem.mockReturnValue('invalid-json');

      const result = authService.getStoredSession();

      expect(result).toBeNull();
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('admin_paseto_token');
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('admin_session');
    });
  });

  describe('session expiry utilities', () => {
    it('should detect when session is expiring soon', () => {
      const soonToExpireSession: AdminSession = {
        isAuthenticated: true,
        pasetoToken: 'mock-token',
        expiresAt: new Date(Date.now() + 5 * 60 * 1000), // 5 minutes from now
        adminInfo: {
          id: '1',
          username: 'admin',
          lastLogin: new Date(),
          permissions: ['admin'],
        },
      };

      localStorageMock.getItem.mockReturnValue(JSON.stringify(soonToExpireSession));

      const result = authService.isSessionExpiringSoon();

      expect(result).toBe(true);
    });

    it('should return false when session is not expiring soon', () => {
      const validSession: AdminSession = {
        isAuthenticated: true,
        pasetoToken: 'mock-token',
        expiresAt: new Date(Date.now() + 30 * 60 * 1000), // 30 minutes from now
        adminInfo: {
          id: '1',
          username: 'admin',
          lastLogin: new Date(),
          permissions: ['admin'],
        },
      };

      localStorageMock.getItem.mockReturnValue(JSON.stringify(validSession));

      const result = authService.isSessionExpiringSoon();

      expect(result).toBe(false);
    });

    it('should calculate correct time until expiry', () => {
      const futureTime = Date.now() + 15 * 60 * 1000; // 15 minutes from now
      const validSession: AdminSession = {
        isAuthenticated: true,
        pasetoToken: 'mock-token',
        expiresAt: new Date(futureTime),
        adminInfo: {
          id: '1',
          username: 'admin',
          lastLogin: new Date(),
          permissions: ['admin'],
        },
      };

      localStorageMock.getItem.mockReturnValue(JSON.stringify(validSession));

      const result = authService.getTimeUntilExpiry();

      // Allow for small timing differences in test execution
      expect(result).toBeGreaterThan(14 * 60 * 1000); // At least 14 minutes
      expect(result).toBeLessThanOrEqual(15 * 60 * 1000); // At most 15 minutes
    });
  });
});