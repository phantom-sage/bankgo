import React from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { motion } from 'framer-motion';
import { useAuth } from '../../hooks/useAuth';

interface ProtectedRouteProps {
  children: React.ReactNode;
  requireAuth?: boolean;
}

/**
 * ProtectedRoute component that handles authentication-based routing
 * - Redirects unauthenticated users to login
 * - Shows loading state during authentication check
 * - Displays session expiry warnings
 */
const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ 
  children, 
  requireAuth = true 
}) => {
  const { isAuthenticated, isLoading, isSessionExpiringSoon, timeUntilExpiry } = useAuth();
  const location = useLocation();

  // Show loading spinner while checking authentication
  if (isLoading) {
    return (
      <div className="min-h-screen bg-admin-background flex items-center justify-center">
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="text-center"
        >
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-admin-primary"></div>
          <p className="mt-4 text-admin-textSecondary">Checking authentication...</p>
        </motion.div>
      </div>
    );
  }

  // Redirect to login if authentication is required but user is not authenticated
  if (requireAuth && !isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // Redirect authenticated users away from login page
  if (!requireAuth && isAuthenticated && location.pathname === '/login') {
    const from = location.state?.from?.pathname || '/';
    return <Navigate to={from} replace />;
  }

  // Show session expiry warning if session is expiring soon
  const SessionExpiryWarning = () => {
    if (!isSessionExpiringSoon || !isAuthenticated) return null;

    const minutes = Math.floor(timeUntilExpiry / (1000 * 60));
    const seconds = Math.floor((timeUntilExpiry % (1000 * 60)) / 1000);

    return (
      <motion.div
        initial={{ opacity: 0, y: -50 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: -50 }}
        className="fixed top-0 left-0 right-0 z-50 bg-yellow-500 text-white px-4 py-2 text-center text-sm font-medium"
      >
        <div className="flex items-center justify-center space-x-2">
          <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
          </svg>
          <span>
            Session expires in {minutes}:{seconds.toString().padStart(2, '0')}. 
            Your session will be automatically refreshed.
          </span>
        </div>
      </motion.div>
    );
  };

  return (
    <>
      <SessionExpiryWarning />
      {children}
    </>
  );
};

export default ProtectedRoute;