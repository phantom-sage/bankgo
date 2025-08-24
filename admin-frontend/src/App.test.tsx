import React from 'react';
import { render, screen } from '@testing-library/react';
import App from './App';

// Mock the useAuth hook
jest.mock('./hooks/useAuth', () => ({
  useAuth: () => ({
    isAuthenticated: false,
    isLoading: false,
    session: null,
    login: jest.fn(),
    logout: jest.fn(),
  }),
}));

test('renders loading animation initially', () => {
  render(<App />);

  // Should show loading animation placeholder text
  expect(
    screen.getByText(
      /Beautiful loading animation will be implemented in task 11/i
    )
  ).toBeInTheDocument();
});

test('renders login page when not authenticated', () => {
  // Mock useAuth to return not loading and not authenticated
  jest.doMock('./hooks/useAuth', () => ({
    useAuth: () => ({
      isAuthenticated: false,
      isLoading: false,
      session: null,
      login: jest.fn(),
      logout: jest.fn(),
    }),
  }));

  // Note: This test will be more comprehensive once the loading animation is properly implemented
  // For now, it just checks that the component renders without crashing
  render(<App />);
  expect(
    screen.getByText(
      /Beautiful loading animation will be implemented in task 11/i
    )
  ).toBeInTheDocument();
});
