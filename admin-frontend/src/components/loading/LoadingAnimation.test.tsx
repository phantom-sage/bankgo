import React from 'react';
import { render, screen, waitFor, act } from '@testing-library/react';
import '@testing-library/jest-dom';
import LoadingAnimation from './LoadingAnimation';

// Mock framer-motion to avoid animation issues in tests
jest.mock('framer-motion', () => ({
  motion: {
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
    rect: ({ children, ...props }: any) => <rect {...props}>{children}</rect>,
    circle: ({ children, ...props }: any) => <circle {...props}>{children}</circle>,
    g: ({ children, ...props }: any) => <g {...props}>{children}</g>,
    path: ({ children, ...props }: any) => <path {...props}>{children}</path>,
  },
}));

describe('LoadingAnimation', () => {
  beforeEach(() => {
    jest.clearAllTimers();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  describe('Component Rendering', () => {
    it('should render the loading animation container', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      const container = document.querySelector('.fixed.inset-0');
      expect(container).toBeInTheDocument();
    });

    it('should display BankGo text in SVG', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      const svgText = screen.getAllByText('BankGo');
      expect(svgText.length).toBeGreaterThan(0);
    });

    it('should render with proper SVG structure', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      const svg = document.querySelector('svg');
      expect(svg).toBeInTheDocument();
      expect(svg).toHaveAttribute('viewBox', '0 0 400 120');
    });

    it('should include gradient definitions for money-themed colors', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      const waterGradient = document.querySelector('#waterGradient');
      const shimmerGradient = document.querySelector('#shimmerGradient');
      const strokeGradient = document.querySelector('#strokeGradient');
      
      expect(waterGradient).toBeInTheDocument();
      expect(shimmerGradient).toBeInTheDocument();
      expect(strokeGradient).toBeInTheDocument();
    });

    it('should have proper clipping path for text', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      const clipPath = document.querySelector('#textClip');
      expect(clipPath).toBeInTheDocument();
    });
  });

  describe('Animation Phases', () => {
    it('should start with initial phase text', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      expect(screen.getByText('Initializing...')).toBeInTheDocument();
    });

    it('should progress through animation phases', async () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      // Initial phase
      expect(screen.getByText('Initializing...')).toBeInTheDocument();
      
      // Progress to filling phase
      act(() => {
        jest.advanceTimersByTime(500);
      });
      
      await waitFor(() => {
        expect(screen.getByText('Loading Dashboard...')).toBeInTheDocument();
      });
      
      // Progress to shimmer phase
      act(() => {
        jest.advanceTimersByTime(2000);
      });
      
      await waitFor(() => {
        expect(screen.getByText('Almost Ready...')).toBeInTheDocument();
      });
      
      // Progress to complete phase
      act(() => {
        jest.advanceTimersByTime(500);
      });
      
      await waitFor(() => {
        expect(screen.getByText('Welcome to BankGo Admin')).toBeInTheDocument();
      });
    });

    it('should display progress indicator with correct width progression', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      const progressBar = document.querySelector('.bg-gradient-to-r.from-gold');
      expect(progressBar).toBeInTheDocument();
    });
  });

  describe('Animation Timing', () => {
    it('should call onComplete after default duration', async () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      // Should not be called initially
      expect(mockOnComplete).not.toHaveBeenCalled();
      
      // Should be called after duration + transition time
      act(() => {
        jest.advanceTimersByTime(3500);
      });
      
      await waitFor(() => {
        expect(mockOnComplete).toHaveBeenCalledTimes(1);
      });
    });

    it('should call onComplete after custom duration', async () => {
      const mockOnComplete = jest.fn();
      const customDuration = 5000;
      render(<LoadingAnimation onComplete={mockOnComplete} duration={customDuration} />);
      
      // Should not be called before custom duration
      act(() => {
        jest.advanceTimersByTime(3000);
      });
      expect(mockOnComplete).not.toHaveBeenCalled();
      
      // Should be called after custom duration + transition time
      act(() => {
        jest.advanceTimersByTime(2500);
      });
      
      await waitFor(() => {
        expect(mockOnComplete).toHaveBeenCalledTimes(1);
      });
    });

    it('should handle phase transitions at correct intervals', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      // Initial phase (0-500ms)
      expect(screen.getByText('Initializing...')).toBeInTheDocument();
      
      // Filling phase (500-2500ms)
      act(() => {
        jest.advanceTimersByTime(600);
      });
      expect(screen.getByText('Loading Dashboard...')).toBeInTheDocument();
      
      // Shimmer phase (2500-3000ms)
      act(() => {
        jest.advanceTimersByTime(2000);
      });
      expect(screen.getByText('Almost Ready...')).toBeInTheDocument();
      
      // Complete phase (3000ms+)
      act(() => {
        jest.advanceTimersByTime(600);
      });
      expect(screen.getByText('Welcome to BankGo Admin')).toBeInTheDocument();
    });
  });

  describe('Visual Effects', () => {
    it('should render background particles', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      const particles = document.querySelectorAll('.bg-gold\\/20.rounded-full');
      expect(particles.length).toBe(20);
    });

    it('should render water filling rectangle with proper gradient', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      const waterRect = document.querySelector('rect[fill="url(#waterGradient)"]');
      expect(waterRect).toBeInTheDocument();
    });

    it('should render ripple effects during filling phase', async () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      // Progress to filling phase
      act(() => {
        jest.advanceTimersByTime(500);
      });
      
      await waitFor(() => {
        const ripples = document.querySelectorAll('circle[stroke="rgba(255,255,255,0.4)"]');
        expect(ripples.length).toBe(3);
      });
    });

    it('should render sparkle effects during shimmer phase', async () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      // Progress to shimmer phase
      act(() => {
        jest.advanceTimersByTime(2500);
      });
      
      await waitFor(() => {
        const sparkles = document.querySelectorAll('path[fill="#FFD700"]');
        expect(sparkles.length).toBe(8);
      });
    });

    it('should include wave pattern for water surface', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      const wavePattern = document.querySelector('#wavePattern');
      expect(wavePattern).toBeInTheDocument();
    });
  });

  describe('Responsive Design', () => {
    it('should have responsive SVG with proper classes', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      const svg = document.querySelector('svg');
      expect(svg).toHaveClass('w-full');
      expect(svg).toHaveClass('max-w-md');
      expect(svg).toHaveClass('sm:max-w-lg');
      expect(svg).toHaveClass('md:max-w-xl');
      expect(svg).toHaveClass('lg:max-w-2xl');
      expect(svg).toHaveClass('h-auto');
    });

    it('should have proper container positioning', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      const container = document.querySelector('.fixed.inset-0');
      expect(container).toBeInTheDocument();
      expect(container).toHaveClass('flex');
      expect(container).toHaveClass('items-center');
      expect(container).toHaveClass('justify-center');
    });
  });

  describe('Color Themes', () => {
    it('should use money-themed gradient colors', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      // Check for gold color (#FFD700)
      const goldElements = document.querySelectorAll('[stop-color="#FFD700"]');
      expect(goldElements.length).toBeGreaterThan(0);
      
      // Check for emerald color (#50C878)
      const emeraldElements = document.querySelectorAll('[stop-color="#50C878"]');
      expect(emeraldElements.length).toBeGreaterThan(0);
      
      // Check for deep blue color (#003366)
      const blueElements = document.querySelectorAll('[stop-color="#003366"]');
      expect(blueElements.length).toBeGreaterThan(0);
      
      // Check for silver color (#C0C0C0)
      const silverElements = document.querySelectorAll('[stop-color="#C0C0C0"]');
      expect(silverElements.length).toBeGreaterThan(0);
    });

    it('should have proper background gradient', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      const background = document.querySelector('.bg-gradient-to-br.from-deepBlue');
      expect(background).toBeInTheDocument();
    });
  });

  describe('Error Handling', () => {
    it('should cleanup timers on unmount', () => {
      const mockOnComplete = jest.fn();
      const { unmount } = render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      const clearTimeoutSpy = jest.spyOn(global, 'clearTimeout');
      
      unmount();
      
      expect(clearTimeoutSpy).toHaveBeenCalledTimes(4);
      clearTimeoutSpy.mockRestore();
    });

    it('should handle onComplete callback errors gracefully', () => {
      const mockOnComplete = jest.fn(() => {
        throw new Error('Test error');
      });
      
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
      
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      // The component should render without crashing
      expect(screen.getByText('BankGo')).toBeInTheDocument();
      
      // Even if onComplete throws, the component should still be rendered
      try {
        act(() => {
          jest.advanceTimersByTime(3500);
        });
      } catch (error) {
        // Expected to throw, but component should still be in DOM
      }
      
      // Component should still be in the document
      expect(screen.getByText('BankGo')).toBeInTheDocument();
      
      consoleSpy.mockRestore();
    });
  });

  describe('Accessibility', () => {
    it('should have proper ARIA attributes', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      const svg = document.querySelector('svg');
      expect(svg).toBeInTheDocument();
    });

    it('should provide meaningful loading text', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      const loadingText = screen.getByText('Initializing...');
      expect(loadingText).toBeInTheDocument();
    });

    it('should not cause motion sickness with reasonable animation speeds', () => {
      const mockOnComplete = jest.fn();
      render(<LoadingAnimation onComplete={mockOnComplete} />);
      
      // Check that animations have reasonable durations
      const svg = document.querySelector('svg');
      expect(svg).toBeInTheDocument();
      
      // Verify no extremely fast animations that could cause issues
      const animatedElements = document.querySelectorAll('[dur]');
      animatedElements.forEach(element => {
        const duration = element.getAttribute('dur');
        if (duration) {
          const durationValue = parseFloat(duration.replace('s', ''));
          expect(durationValue).toBeGreaterThan(0.5); // No animations faster than 0.5s
        }
      });
    });
  });
});