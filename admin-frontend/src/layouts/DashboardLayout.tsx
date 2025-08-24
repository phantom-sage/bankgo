import React from 'react';

interface DashboardLayoutProps {
  children: React.ReactNode;
}

const DashboardLayout: React.FC<DashboardLayoutProps> = ({ children }) => {
  return (
    <div className="min-h-screen bg-admin-background">
      {/* Sidebar Navigation - to be implemented in task 13 */}
      <div className="flex">
        <aside className="w-64 bg-admin-surface shadow-lg">
          <div className="p-4">
            <h2 className="text-xl font-bold text-admin-primary">
              BankGo Admin
            </h2>
            <p className="text-sm text-admin-textSecondary">
              Dashboard navigation will be implemented in task 13
            </p>
          </div>
        </aside>

        {/* Main Content Area */}
        <main className="flex-1 p-6">{children}</main>
      </div>
    </div>
  );
};

export default DashboardLayout;
