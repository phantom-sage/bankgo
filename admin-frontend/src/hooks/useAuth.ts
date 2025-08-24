import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { AdminSession, LoginCredentials, ApiResponse } from '../types';
import { authService } from '../services/authService';

interface UseAuthReturn {
  isAuthenticated: boolean;
  isLoading: boolean;
  session: AdminSession | null;
  login: (username: string, password: string) => Promise<ApiResponse<AdminSession>>;
  logout: () => Promise<void>;
  refreshSession: () => Promise<void>;
  isSessionExpiringSoon: boolean;
  timeUntilExpiry: number;
}

export const useAuth = (): UseAuthReturn => {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [session, setSession] = useState<AdminSession | null>(null);
  const [isSessionExpiringSoon, setIsSessionExpiringSoon] = useState(false);
  const [timeUntilExpiry, setTimeUntilExpiry] = useState(0);
  const navigate = useNavigate();

  // Check session expiry status periodically
  useEffect(() => {
    const checkExpiryStatus = () => {
      if (session) {
        const expiringSoon = authService.isSessionExpiringSoon();
        const timeRemaining = authService.getTimeUntilExpiry();
        
        setIsSessionExpiringSoon(expiringSoon);
        setTimeUntilExpiry(timeRemaining);

        // Auto-logout if session has expired
        if (timeRemaining <= 0) {
          handleLogout();
        }
      }
    };

    // Check immediately and then every 30 seconds
    checkExpiryStatus();
    const interval = setInterval(checkExpiryStatus, 30000);

    return () => clearInterval(interval);
  }, [session]);

  // Initialize authentication state on mount
  useEffect(() => {
    const initializeAuth = async () => {
      try {
        // First check if we have a stored session
        const storedSession = authService.getStoredSession();
        
        if (storedSession) {
          // Validate the session with the backend
          const result = await authService.validateSession();
          
          if (result.success && result.data) {
            setSession(result.data);
            setIsAuthenticated(true);
          } else {
            // Session is invalid, clear it
            setSession(null);
            setIsAuthenticated(false);
          }
        } else {
          setSession(null);
          setIsAuthenticated(false);
        }
      } catch (error) {
        console.error('Error initializing auth:', error);
        setSession(null);
        setIsAuthenticated(false);
      } finally {
        setIsLoading(false);
      }
    };

    initializeAuth();
  }, []);

  const login = useCallback(async (username: string, password: string): Promise<ApiResponse<AdminSession>> => {
    const credentials: LoginCredentials = { username, password };
    
    try {
      const result = await authService.login(credentials);
      
      if (result.success && result.data) {
        setSession(result.data);
        setIsAuthenticated(true);
        
        // Navigate to dashboard on successful login
        navigate('/', { replace: true });
      }
      
      return result;
    } catch (error) {
      return {
        success: false,
        error: 'Login failed',
        message: error instanceof Error ? error.message : 'An unexpected error occurred',
      };
    }
  }, [navigate]);

  const handleLogout = useCallback(async () => {
    try {
      await authService.logout();
    } catch (error) {
      console.error('Logout error:', error);
    } finally {
      setSession(null);
      setIsAuthenticated(false);
      setIsSessionExpiringSoon(false);
      setTimeUntilExpiry(0);
      
      // Navigate to login page
      navigate('/login', { replace: true });
    }
  }, [navigate]);

  const logout = useCallback(async (): Promise<void> => {
    await handleLogout();
  }, [handleLogout]);

  const refreshSession = useCallback(async (): Promise<void> => {
    try {
      const result = await authService.validateSession();
      
      if (result.success && result.data) {
        setSession(result.data);
        setIsAuthenticated(true);
      } else {
        // Session refresh failed, logout user
        await handleLogout();
      }
    } catch (error) {
      console.error('Session refresh error:', error);
      await handleLogout();
    }
  }, [handleLogout]);

  // Auto-refresh session when it's about to expire
  useEffect(() => {
    if (isSessionExpiringSoon && session && !isLoading) {
      refreshSession();
    }
  }, [isSessionExpiringSoon, session, isLoading, refreshSession]);

  return {
    isAuthenticated,
    isLoading,
    session,
    login,
    logout,
    refreshSession,
    isSessionExpiringSoon,
    timeUntilExpiry,
  };
};
