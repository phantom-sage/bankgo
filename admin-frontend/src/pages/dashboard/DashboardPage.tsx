import React from 'react';

const DashboardPage: React.FC = () => {
  return (
    <div className="space-y-6">
      <div className="bg-admin-surface rounded-lg shadow p-6">
        <h1 className="text-3xl font-bold text-admin-text mb-4">
          Admin Dashboard
        </h1>
        <p className="text-admin-textSecondary">
          Welcome to the BankGo Admin Dashboard. This is a placeholder page that
          will be implemented with full functionality in later tasks.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div className="bg-admin-surface rounded-lg shadow p-6">
          <h3 className="text-lg font-semibold text-admin-text mb-2">
            System Health
          </h3>
          <p className="text-admin-textSecondary">Task 15</p>
        </div>

        <div className="bg-admin-surface rounded-lg shadow p-6">
          <h3 className="text-lg font-semibold text-admin-text mb-2">
            User Management
          </h3>
          <p className="text-admin-textSecondary">Task 14</p>
        </div>

        <div className="bg-admin-surface rounded-lg shadow p-6">
          <h3 className="text-lg font-semibold text-admin-text mb-2">
            Transactions
          </h3>
          <p className="text-admin-textSecondary">Task 17</p>
        </div>

        <div className="bg-admin-surface rounded-lg shadow p-6">
          <h3 className="text-lg font-semibold text-admin-text mb-2">
            Database
          </h3>
          <p className="text-admin-textSecondary">Task 16</p>
        </div>
      </div>
    </div>
  );
};

export default DashboardPage;
