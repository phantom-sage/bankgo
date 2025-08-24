import React from 'react';

interface AuthLayoutProps {
  children: React.ReactNode;
}

const AuthLayout: React.FC<AuthLayoutProps> = ({ children }) => {
  return (
    <div className="min-h-screen bg-gradient-to-br from-admin-primary to-deepBlue flex items-center justify-center">
      <div className="bg-admin-surface rounded-lg shadow-xl p-8 w-full max-w-md">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-admin-primary">BankGo</h1>
          <p className="text-admin-textSecondary">Admin Dashboard</p>
        </div>
        {children}
      </div>
    </div>
  );
};

export default AuthLayout;
