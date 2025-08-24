/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./src/**/*.{js,jsx,ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Money-themed colors for the admin dashboard
        gold: '#FFD700',
        emerald: '#50C878',
        deepBlue: '#003366',
        silver: '#C0C0C0',
        // Admin dashboard specific colors
        admin: {
          primary: '#1e40af',
          secondary: '#64748b',
          success: '#10b981',
          warning: '#f59e0b',
          error: '#ef4444',
          background: '#f8fafc',
          surface: '#ffffff',
          text: '#1e293b',
          textSecondary: '#64748b',
        }
      },
      animation: {
        'water-fill': 'waterFill 2s ease-in-out forwards',
        'shimmer': 'shimmer 1.5s ease-in-out infinite',
        'fade-in': 'fadeIn 0.5s ease-in-out',
        'slide-in': 'slideIn 0.3s ease-out',
      },
      keyframes: {
        waterFill: {
          '0%': { height: '0%' },
          '100%': { height: '100%' }
        },
        shimmer: {
          '0%, 100%': { opacity: 1 },
          '50%': { opacity: 0.7 }
        },
        fadeIn: {
          '0%': { opacity: 0 },
          '100%': { opacity: 1 }
        },
        slideIn: {
          '0%': { transform: 'translateX(-100%)' },
          '100%': { transform: 'translateX(0)' }
        }
      }
    },
  },
  plugins: [],
}