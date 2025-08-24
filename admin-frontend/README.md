# BankGo Admin Dashboard

A modern, responsive admin dashboard for the BankGo banking system built with React, TypeScript, and Tailwind CSS.

## Features

- 🎨 Beautiful, responsive UI with Tailwind CSS
- 🔐 Secure authentication with PASETO tokens
- 📊 Real-time system monitoring and alerts
- 👥 Comprehensive user management
- 💰 Transaction monitoring and management
- 🗄️ Direct database operations
- 🌊 Stunning water-fill loading animation
- ⚡ Real-time notifications via WebSocket

## Tech Stack

- **React 19** - Modern React with latest features
- **TypeScript** - Type-safe development
- **Tailwind CSS** - Utility-first CSS framework
- **Framer Motion** - Smooth animations and transitions
- **React Router** - Client-side routing
- **WebSocket** - Real-time communication

## Project Structure

```
src/
├── components/          # Reusable UI components
│   ├── ui/             # Basic UI components
│   ├── forms/          # Form components
│   ├── charts/         # Chart components
│   └── loading/        # Loading components
├── pages/              # Page components
│   ├── auth/           # Authentication pages
│   ├── dashboard/      # Dashboard pages
│   ├── users/          # User management pages
│   ├── transactions/   # Transaction pages
│   ├── system/         # System monitoring pages
│   └── database/       # Database management pages
├── layouts/            # Layout components
├── hooks/              # Custom React hooks
├── services/           # API and WebSocket services
├── types/              # TypeScript type definitions
├── utils/              # Utility functions
└── App.tsx            # Main application component
```

## Getting Started

### Prerequisites

- Node.js 16+ 
- npm or yarn

### Installation

1. Install dependencies:
   ```bash
   npm install
   ```

2. Start the development server:
   ```bash
   npm start
   ```

3. Open [http://localhost:3000](http://localhost:3000) to view it in the browser.

### Available Scripts

- `npm start` - Start development server
- `npm run build` - Build for production
- `npm test` - Run tests once
- `npm run test:watch` - Run tests in watch mode
- `npm run test:coverage` - Run tests with coverage report
- `npm run lint` - Check code quality
- `npm run lint:fix` - Fix linting issues
- `npm run format` - Format code with Prettier
- `npm run format:check` - Check code formatting

## Development Status

This project is being developed using spec-driven development methodology. Current implementation status:

- ✅ Task 10: Project structure setup (COMPLETED)
- ⏳ Task 11: Loading animation component (PENDING)
- ⏳ Task 12: Authentication service and login (PENDING)
- ⏳ Task 13: Dashboard layout and navigation (PENDING)
- ⏳ Task 14: User management interface (PENDING)
- ⏳ Task 15: System monitoring dashboard (PENDING)
- ⏳ Task 16: Database management interface (PENDING)
- ⏳ Task 17: Transaction management interface (PENDING)
- ⏳ Task 18: Real-time notification system (PENDING)

## Environment Variables

Create a `.env` file in the root directory:

```env
REACT_APP_API_URL=http://localhost:8081/api/admin
REACT_APP_WS_URL=ws://localhost:8081/api/admin/ws
```

## API Integration

The frontend communicates with the BankGo Admin API server running on port 8081. Key endpoints:

- `POST /api/admin/auth/login` - Admin authentication
- `GET /api/admin/health` - System health metrics
- `GET /api/admin/users` - User management
- `GET /api/admin/transactions` - Transaction data
- `WebSocket /api/admin/ws/notifications` - Real-time updates

## Design System

### Colors

The dashboard uses a money-themed color palette:

- **Gold** (#FFD700) - Premium features and highlights
- **Emerald** (#50C878) - Success states and positive metrics
- **Deep Blue** (#003366) - Primary branding and navigation
- **Silver** (#C0C0C0) - Secondary elements and borders

### Admin Theme

- **Primary** - Professional blue for main actions
- **Success** - Green for positive states
- **Warning** - Amber for caution states
- **Error** - Red for error states
- **Background** - Light gray for main background
- **Surface** - White for content areas

## Testing

The project includes comprehensive testing setup:

- **Unit Tests** - Component and utility function tests
- **Integration Tests** - API communication and user flows
- **E2E Tests** - Complete user workflows (planned)

Run tests with:
```bash
npm test
```

## Contributing

1. Follow the existing code style and conventions
2. Write tests for new features
3. Update documentation as needed
4. Use TypeScript for type safety
5. Follow the component structure and naming conventions

## License

This project is part of the BankGo banking system and is proprietary software.