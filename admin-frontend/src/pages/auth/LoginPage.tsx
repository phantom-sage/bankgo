import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { useAuth } from '../../hooks/useAuth';
import { LoginCredentials } from '../../types';

interface FormErrors {
  username?: string;
  password?: string;
  general?: string;
}

const LoginPage: React.FC = () => {
  const { login, isLoading } = useAuth();
  const [credentials, setCredentials] = useState<LoginCredentials>({
    username: '',
    password: '',
  });
  const [errors, setErrors] = useState<FormErrors>({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [loginAttempts, setLoginAttempts] = useState(0);
  const [lockoutTime, setLockoutTime] = useState<Date | null>(null);
  const [countdown, setCountdown] = useState(0);

  // Handle lockout countdown
  useEffect(() => {
    if (lockoutTime) {
      const timer = setInterval(() => {
        const now = new Date().getTime();
        const lockout = lockoutTime.getTime();
        const remaining = Math.max(0, Math.ceil((lockout - now) / 1000));
        
        setCountdown(remaining);
        
        if (remaining === 0) {
          setLockoutTime(null);
          setLoginAttempts(0);
          clearInterval(timer);
        }
      }, 1000);

      return () => clearInterval(timer);
    }
  }, [lockoutTime]);

  const validateForm = (): boolean => {
    const newErrors: FormErrors = {};

    if (!credentials.username.trim()) {
      newErrors.username = 'Username is required';
    } else if (credentials.username.length < 2) {
      newErrors.username = 'Username must be at least 2 characters';
    }

    if (!credentials.password) {
      newErrors.password = 'Password is required';
    } else if (credentials.password.length < 3) {
      newErrors.password = 'Password must be at least 3 characters';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleInputChange = (field: keyof LoginCredentials, value: string) => {
    setCredentials(prev => ({ ...prev, [field]: value }));
    
    // Clear field-specific error when user starts typing
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: undefined }));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    // Check if locked out
    if (lockoutTime && new Date() < lockoutTime) {
      return;
    }

    // Clear general error
    setErrors(prev => ({ ...prev, general: undefined }));

    if (!validateForm()) {
      return;
    }

    setIsSubmitting(true);

    try {
      const result = await login(credentials.username, credentials.password);
      
      if (!result.success) {
        const newAttempts = loginAttempts + 1;
        setLoginAttempts(newAttempts);

        // Lock out after 3 failed attempts
        if (newAttempts >= 3) {
          const lockout = new Date();
          lockout.setMinutes(lockout.getMinutes() + 10); // 10 minute lockout
          setLockoutTime(lockout);
          setErrors({ general: 'Too many failed attempts. Account locked for 10 minutes.' });
        } else {
          setErrors({ 
            general: result.message || 'Invalid credentials',
          });
        }
      } else {
        // Reset attempts on successful login
        setLoginAttempts(0);
        setLockoutTime(null);
      }
    } catch (error) {
      setErrors({ 
        general: 'An unexpected error occurred. Please try again.',
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const isLocked = lockoutTime && new Date() < lockoutTime;

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
      className="space-y-6"
    >
      <div className="text-center">
        <h2 className="text-2xl font-semibold text-admin-text">Sign In</h2>
        <p className="text-admin-textSecondary mt-2">
          Access the BankGo Admin Dashboard
        </p>
      </div>

      {/* Lockout Warning */}
      {isLocked && (
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          className="bg-red-50 border border-red-200 rounded-md p-4"
        >
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M10 1.944A11.954 11.954 0 012.166 5C2.056 5.649 2 6.319 2 7c0 5.225 3.34 9.67 8 11.317C14.66 16.67 18 12.225 18 7c0-.682-.057-1.35-.166-2.001A11.954 11.954 0 0110 1.944zM11 14a1 1 0 11-2 0 1 1 0 012 0zm0-7a1 1 0 10-2 0v3a1 1 0 102 0V7z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">
                Account Temporarily Locked
              </h3>
              <div className="mt-2 text-sm text-red-700">
                <p>Too many failed login attempts. Please wait {countdown} seconds before trying again.</p>
              </div>
            </div>
          </div>
        </motion.div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Username Field */}
        <div>
          <label className="block text-sm font-medium text-admin-text mb-2">
            Username
          </label>
          <div className="relative">
            <input
              type="text"
              value={credentials.username}
              onChange={(e) => handleInputChange('username', e.target.value)}
              placeholder="Enter username (default: admin)"
              className={`w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 transition-colors ${
                errors.username
                  ? 'border-red-300 focus:ring-red-500 focus:border-red-500'
                  : 'border-gray-300 focus:ring-admin-primary focus:border-admin-primary'
              } ${isLocked ? 'opacity-50 cursor-not-allowed' : ''}`}
              disabled={isSubmitting || isLocked}
              autoComplete="username"
            />
            {errors.username && (
              <motion.p
                initial={{ opacity: 0, y: -10 }}
                animate={{ opacity: 1, y: 0 }}
                className="mt-1 text-sm text-red-600"
              >
                {errors.username}
              </motion.p>
            )}
          </div>
        </div>

        {/* Password Field */}
        <div>
          <label className="block text-sm font-medium text-admin-text mb-2">
            Password
          </label>
          <div className="relative">
            <input
              type={showPassword ? 'text' : 'password'}
              value={credentials.password}
              onChange={(e) => handleInputChange('password', e.target.value)}
              placeholder="Enter password (default: admin)"
              className={`w-full px-3 py-2 pr-10 border rounded-md focus:outline-none focus:ring-2 transition-colors ${
                errors.password
                  ? 'border-red-300 focus:ring-red-500 focus:border-red-500'
                  : 'border-gray-300 focus:ring-admin-primary focus:border-admin-primary'
              } ${isLocked ? 'opacity-50 cursor-not-allowed' : ''}`}
              disabled={isSubmitting || isLocked}
              autoComplete="current-password"
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute inset-y-0 right-0 pr-3 flex items-center"
              disabled={isLocked}
            >
              <svg
                className={`h-5 w-5 ${isLocked ? 'text-gray-300' : 'text-gray-400 hover:text-gray-600'}`}
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                {showPassword ? (
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.878 9.878L3 3m6.878 6.878L21 21" />
                ) : (
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                )}
              </svg>
            </button>
            {errors.password && (
              <motion.p
                initial={{ opacity: 0, y: -10 }}
                animate={{ opacity: 1, y: 0 }}
                className="mt-1 text-sm text-red-600"
              >
                {errors.password}
              </motion.p>
            )}
          </div>
        </div>

        {/* General Error */}
        {errors.general && (
          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            className="bg-red-50 border border-red-200 rounded-md p-3"
          >
            <div className="flex">
              <div className="flex-shrink-0">
                <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                </svg>
              </div>
              <div className="ml-3">
                <p className="text-sm text-red-800">{errors.general}</p>
              </div>
            </div>
          </motion.div>
        )}

        {/* Submit Button */}
        <motion.button
          type="submit"
          disabled={isSubmitting || isLocked || isLoading}
          whileHover={!isSubmitting && !isLocked ? { scale: 1.02 } : {}}
          whileTap={!isSubmitting && !isLocked ? { scale: 0.98 } : {}}
          className={`w-full py-2 px-4 rounded-md font-medium transition-colors ${
            isSubmitting || isLocked || isLoading
              ? 'bg-gray-400 cursor-not-allowed'
              : 'bg-admin-primary hover:bg-blue-700 focus:ring-2 focus:ring-admin-primary focus:ring-offset-2'
          } text-white`}
        >
          {isSubmitting ? (
            <div className="flex items-center justify-center">
              <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              Signing In...
            </div>
          ) : isLocked ? (
            `Locked (${countdown}s)`
          ) : (
            'Sign In'
          )}
        </motion.button>
      </form>

      {/* Login Attempts Warning */}
      {loginAttempts > 0 && loginAttempts < 3 && !isLocked && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="text-center text-sm text-yellow-600"
        >
          {loginAttempts === 1 && 'Invalid credentials. 2 attempts remaining.'}
          {loginAttempts === 2 && 'Invalid credentials. 1 attempt remaining before lockout.'}
        </motion.div>
      )}

      {/* Default Credentials Hint */}
      <div className="text-center text-xs text-admin-textSecondary">
        <p>Default credentials: admin / admin</p>
        <p>Session expires after 1 hour</p>
      </div>
    </motion.div>
  );
};

export default LoginPage;
