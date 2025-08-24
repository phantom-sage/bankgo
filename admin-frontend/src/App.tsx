import React from 'react';
import {
  BrowserRouter as Router,
  Routes,
  Route,
  Navigate,
} from 'react-router-dom';
import { motion } from 'framer-motion';
import './App.css';

// Layout Components (to be implemented in later tasks)
import DashboardLayout from './layouts/DashboardLayout';
import AuthLayout from './layouts/AuthLayout';

// Page Components (to be implemented in later tasks)
import LoginPage from './pages/auth/LoginPage';
import DashboardPage from './pages/dashboard/DashboardPage';
import UsersPage from './pages/users/UsersPage';
import TransactionsPage from './pages/transactions/TransactionsPage';
import SystemPage from './pages/system/SystemPage';
import DatabasePage from './pages/database/DatabasePage';

// Loading Animation Component (implemented in task 11)
import LoadingAnimation from './components/loading/LoadingAnimation';

// Authentication Components
import ProtectedRoute from './components/auth/ProtectedRoute';

// Hooks
import { useAuth } from './hooks/useAuth';

function App() {
  const { isAuthenticated } = useAuth();
  const [showLoading, setShowLoading] = React.useState(true);

  const handleLoadingComplete = () => {
    // For testing, let's show the animation again after 2 seconds
    setTimeout(() => {
      setShowLoading(true);
    }, 2000);
    setShowLoading(false);
  };

  // Always show loading animation for testing
  if (showLoading) {
    return <LoadingAnimation onComplete={handleLoadingComplete} />;
  }

  return (
    <Router>
      <motion.div
        className="min-h-screen bg-admin-background"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.5 }}
      >
        <Routes>
          {/* Public Routes */}
          <Route
            path="/login"
            element={
              <ProtectedRoute requireAuth={false}>
                <AuthLayout>
                  <LoginPage />
                </AuthLayout>
              </ProtectedRoute>
            }
          />

          {/* Protected Routes */}
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <DashboardLayout>
                  <Routes>
                    <Route index element={<DashboardPage />} />
                    <Route path="dashboard" element={<DashboardPage />} />
                    <Route path="users/*" element={<UsersPage />} />
                    <Route
                      path="transactions/*"
                      element={<TransactionsPage />}
                    />
                    <Route path="system/*" element={<SystemPage />} />
                    <Route path="database/*" element={<DatabasePage />} />
                  </Routes>
                </DashboardLayout>
              </ProtectedRoute>
            }
          />

          {/* Catch all route */}
          <Route
            path="*"
            element={<Navigate to={isAuthenticated ? '/' : '/login'} replace />}
          />
        </Routes>
      </motion.div>
    </Router>
  );
}

export default App;
